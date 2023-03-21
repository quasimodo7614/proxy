package queue

import "testing"

func TestNew(t *testing.T) {
	q := New(3)
	q.Add("1")
	q.Add("2")
	q.Add("3")
	t.Log(q.GetMsg())
}
