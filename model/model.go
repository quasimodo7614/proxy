package model

type Msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AiReq struct {
	Model    string `json:"model"`
	Messages []Msg  `json:"messages"`
}

type Resp struct {
	Cont string `json:"cont"`
}
type Req struct {
	Msg []Msg
}

type User struct {
	User string `json:"User"`
	Pwd  string `json:"Pwd"`
}

type GenTokenResp struct {
	Jwt string `json:"Jwt"`
}

type Url struct {
	Url string `json:"url"`
}
type ImageResp struct {
	Data []Url `json:"data"`
}

type ImageReq struct {
	Msg string `json:"msg"`
}

type AiImageReq struct {
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}
