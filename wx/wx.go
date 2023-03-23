package wx

import (
	"encoding/xml"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

// WXTextMsg 微信文本消息结构体
type WXTextMsg struct {
	ToUserName   string
	FromUserName string
	CreateTime   int64
	MsgType      string
	Content      string
	MsgId        int64
}

// WXRepTextMsg 微信回复文本消息结构体
type WXRepTextMsg struct {
	ToUserName   string
	FromUserName string
	CreateTime   int64
	MsgType      string
	Content      string
	// 若不标记XMLName, 则解析后的xml名为该结构体的名称
	XMLName xml.Name `xml:"xml"`
}

type AccessTokenResponse struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"`
}

type AccessTokenErrorResponse struct {
	Errcode float64
	Errmsg  string
}

const (
	token               = "zzedjsjh"
	appID               = "wx10ad4733ca90bf2b"
	appSecret           = "a937c8ddbe4ce3351c1b77238e68fbff"
	accessTokenFetchUrl = "https://api.weixin.qq.com/cgi-bin/token"
)

// todo : 签名验证

func Wx() gin.HandlerFunc {
	return func(c *gin.Context) {
		echostr := c.Query("echostr")
		log.Println("echo str is: ", echostr)
		c.AbortWithStatus(200)
		c.Writer.Write([]byte(echostr))
	}

}

// WXMsgReceive 微信消息接收
func WXMsgReceive(c *gin.Context) {
	var textMsg WXTextMsg
	err := c.ShouldBindXML(&textMsg)
	if err != nil {
		log.Printf("[消息接收] - XML数据包解析失败: %v\n", err)
		return
	}

	log.Printf("[消息接收] - 收到消息, 消息类型为: %s, 消息内容为: %s\n", textMsg.MsgType, textMsg.Content)
	// 对接收的消息进行被动回复
	WXMsgReply(c, textMsg.ToUserName, textMsg.FromUserName)
}

// WXMsgReply 微信消息回复
func WXMsgReply(c *gin.Context, fromUser, toUser string) {
	repTextMsg := WXRepTextMsg{
		ToUserName:   toUser,
		FromUserName: fromUser,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      "由于 openai 接口经常超过 5s，建议去我们的官网访问: http://82.156.167.18:2023/", // 这里要立刻返回，否则微信会自动重试
	}

	msg, err := xml.Marshal(&repTextMsg)
	if err != nil {
		log.Printf("[消息回复] - 将对象进行XML编码出错: %v\n", err)
		return
	}
	_, _ = c.Writer.Write(msg)
}
