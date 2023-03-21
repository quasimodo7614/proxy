package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"proxy/model"
	"proxy/queue"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/tidwall/gjson"
)

// Chat 所有相关参数的聚合
type Chat struct {
	JwtSecret     string
	DisAbleJwt    bool
	Queue         *queue.Queue
	OpenAIUrl     string
	SocksProxyUrl string
	APIkey        string
}

func NewChat() *Chat {
	chat := &Chat{}
	if s := os.Getenv("JWT_Secret"); s != "" {
		chat.JwtSecret = s
	} else {
		chat.JwtSecret = "123456"
	}
	if s := os.Getenv("Disable_Jwt"); s == "true" {
		chat.DisAbleJwt = true
	}
	msgLen := 10
	if i, _ := strconv.Atoi(os.Getenv("Msg_Array_Num")); i > 0 {
		msgLen = i
	}
	chat.Queue = queue.New(msgLen)

	chat.OpenAIUrl = "https://api.openai.com/v1/chat/completions"
	if s := os.Getenv("OPENAI_URL"); s != "" {
		chat.OpenAIUrl = s
	}

	chat.SocksProxyUrl = "127.0.0.1:1080"
	if s := os.Getenv("PROXY_HOST"); s != "" {
		chat.SocksProxyUrl = s
	}

	if s := os.Getenv("OPENAI_API_KEY"); s != "" {
		chat.APIkey = s
	}

	return chat
}

func (chat *Chat) ginChat() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("read body err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		req := &model.Req{}
		err = json.Unmarshal(bodyBytes, req)
		if err != nil {
			log.Println("unmarshal err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		resp := &model.Resp{}
		respB, err := chat.dopost(req.Cont)
		if err != nil {
			log.Println("do post  err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		resp.Cont = string(respB)
		c.JSON(http.StatusOK, resp)
		return
	}

}

func (chat *Chat) JwtCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if chat.DisAbleJwt {
			return
		}
		tokenString := c.GetHeader("Authorization")
		var hmacSampleSecret = []byte(chat.JwtSecret)
		//前面例子生成的token
		token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
			return hmacSampleSecret, nil
		})

		if err != nil {
			log.Println("ParseWithClaims err: ", err)
			c.AbortWithStatusJSON(400, err.Error())
			return
		}
		if !token.Valid {
			c.AbortWithStatusJSON(400, err.Error())
			return
		}
		log.Println("token valid")
		c.Next()
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func getSvc() string {
	if s := os.Getenv("SVC_ADDR"); s != "" {
		return s
	}
	return ":9298"
}

func main() {

	r := gin.Default()
	chat := NewChat()
	r.Use(chat.JwtCheck(), Cors())

	r.POST("/chat", chat.ginChat())
	if err := r.Run(getSvc()); err != nil {
		log.Fatal("start err: ", err)
	}

}

func (chat *Chat) dopost(content string) ([]byte, error) {
	client, err := chat.NewClientFromEnv()
	if err != nil {
		log.Println("new client err: ", err.Error())
		return nil, err
	}
	reqRaw := model.AiReq{
		Model: "gpt-3.5-turbo",
		Messages: []model.Msg{
			{
				Role:    "user",
				Content: content,
			},
		},
	}
	reqRaw.Messages = append(reqRaw.Messages, chat.Queue.GetMsg()...)
	b, err := json.Marshal(reqRaw)
	if err != nil {
		log.Println(err, " marshal err")
		return nil, err
	}
	log.Println("req raw is: ", string(b))
	req, err := http.NewRequest("POST", chat.OpenAIUrl, bytes.NewReader(b))
	if err != nil {
		log.Println(err, " new request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+chat.APIkey)
	//fmt.Println("req header is: ", req.Header)
	resp, err := client.Do(req)
	if err != nil {
		log.Println("client req err:", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	log.Println("resp status:", resp.StatusCode)
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//fmt.Println("resp is:", string(out))
	respCont := gjson.Get(string(out), "choices.0.message.content").String()
	//log.Println("respCont is:", respCont)
	chat.Queue.Add(respCont)
	return out, nil

}

// NewClientFromEnv  example that creates an http client that leverages a SOCKS5 proxy and a DialContext
func (chat *Chat) NewClientFromEnv() (*http.Client, error) {
	proxyHost := chat.SocksProxyUrl

	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	var dialContext func(ctx context.Context, network, address string) (net.Conn, error)

	if proxyHost != "" {
		log.Println("proxy host: ", proxyHost)
		dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, baseDialer)
		if err != nil {
			log.Println("new sockts 5 err:", err.Error())
			return nil, err
		}
		if contextDialer, ok := dialSocksProxy.(proxy.ContextDialer); ok {
			dialContext = contextDialer.DialContext
		} else {
			return nil, errors.New("Failed type assertion to DialContext")
		}
		log.Println("proxy host ok")
	} else {
		log.Println("default contex")
		dialContext = (baseDialer).DialContext
	}
	httpClient := newClient(dialContext)
	return httpClient, nil
}

func newClient(dialContext func(ctx context.Context, network, address string) (net.Conn, error)) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
}
