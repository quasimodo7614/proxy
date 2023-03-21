package queue

import "proxy/model"

// special queue for chart gpt

type Queue struct {
	Size int
	Msg  []model.Msg
}

func New(size int) *Queue {
	return &Queue{
		Size: size,
	}

}
func (q *Queue) Add(s string) {
	msg := model.Msg{
		Role:    "assistant",
		Content: s,
	}
	if len(q.Msg) >= q.Size { // 已经满了，去除第一个，加在最后面
		q.Msg = q.Msg[1:q.Size]
	}
	q.Msg = append(q.Msg, msg)
}

func (q *Queue) GetMsg() []model.Msg {
	return q.Msg
}
