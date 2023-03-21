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
	Cont string `json:"cont"`
}
