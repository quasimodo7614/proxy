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
	"proxy/user"
	"proxy/wx"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// Chat 所有相关参数的聚合
type Chat struct {
	Queue         *queue.Queue
	ChatUrl       string
	ImageUrl      string
	SocksProxyUrl string
	APIkey        string
	Continue      bool
}

var qps = map[int]time.Time{}

func NewChat() *Chat {
	chat := &Chat{}
	//msgLen := 100
	//if i, _ := strconv.Atoi(os.Getenv("Msg_Array_Num")); i > 0 {
	//	msgLen = i
	//}
	//chat.Queue = queue.New(msgLen)

	chat.ChatUrl = "https://api.openai.com/v1/chat/completions"
	if s := os.Getenv("ChatUrl"); s != "" {
		chat.ChatUrl = s
	}
	chat.ImageUrl = "https://api.openai.com/v1/images/generations"
	if s := os.Getenv("ImageUrl"); s != "" {
		chat.ChatUrl = s
	}

	chat.SocksProxyUrl = "127.0.0.1:1080"
	if s := os.Getenv("PROXY_HOST"); s != "" {
		chat.SocksProxyUrl = s
	}

	if s := os.Getenv("OPENAI_API_KEY"); s != "" {
		chat.APIkey = s
	}

	if s := os.Getenv("Continue_Chat"); s == "true" {
		chat.Continue = true
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
		reqRaw := model.AiReq{
			Model: "gpt-3.5-turbo",
		}
		reqRaw.Messages = req.Msg
		respB, err := chat.dopost(reqRaw, chat.ChatUrl)
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

func (chat *Chat) ginImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("read body err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		req := &model.ImageReq{}
		err = json.Unmarshal(bodyBytes, req)
		if err != nil {
			log.Println("unmarshal err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		resp := &model.ImageResp{}
		reqRaw := &model.AiImageReq{
			Prompt: req.Msg,
			N:      1, // 暂时写死1
			Size:   "1024x1024",
		}
		respB, err := chat.dopost(reqRaw, chat.ImageUrl)
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

// 需要限制一个 qps ，不然答案和问题对不上
func qpsmiddleware() gin.HandlerFunc {
	if qps == nil {
		qps = map[int]time.Time{}
	}
	return func(c *gin.Context) {
		last, ok := qps[1]
		// 说明上次请求时间是 10s 以内，并且还没响应
		if ok && last.Add(time.Second*10).After(time.Now()) {
			c.AbortWithStatusJSON(400, "正在处理其他问题，请耐心等候10s左右")
			return
		}
		qps[1] = time.Now()
		c.Next()
		delete(qps, 1)
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
	r.Use(qpsmiddleware(), Cors())

	r.POST("/chat", chat.ginChat())
	r.POST("/image", chat.ginImage())
	r.POST("/user", user.User())
	r.GET("/wx", wx.Wx())
	r.POST("/wx", wx.WXMsgReceive)
	if err := r.Run(getSvc()); err != nil {
		log.Fatal("start err: ", err)
	}

}

// assistants 是用来做连续对话的，是之前问题的答案。
func (chat *Chat) dopost(msg interface{}, url string) ([]byte, error) {
	client, err := chat.NewClientFromEnv()
	if err != nil {
		log.Println("new client err: ", err.Error())
		return nil, err
	}

	b, err := json.Marshal(msg)
	if err != nil {
		log.Println(err, " marshal err")
		return nil, err
	}
	log.Println("req raw is: ", string(b))
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
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
	////fmt.Println("resp is:", string(out))
	//respCont := gjson.Get(string(out), "choices.0.message.content").String()
	////log.Println("respCont is:", respCont)
	//chat.Queue.Add(respCont)
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
