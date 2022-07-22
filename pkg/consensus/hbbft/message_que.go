package hbbft

import (
	"sync"
)

// MessageTuple holds the payload of the message along with the identifier of
// the receiver node.
type MessageTuple struct {
	To      string
	Payload interface{}
}

type messageQue struct {
	que  []MessageTuple
	lock sync.RWMutex
}

func newMessageQue() *messageQue {
	return &messageQue{
		que: []MessageTuple{},
	}
}

func (q *messageQue) addMessage(msg interface{}, to string) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.que = append(q.que, MessageTuple{to, msg})
}

func (q *messageQue) addMessages(msgs ...MessageTuple) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.que = append(q.que, msgs...)
}

func (q *messageQue) addQue(que *messageQue) {
	newQue := make([]MessageTuple, len(q.que)+que.len())
	copy(newQue, q.messages())
	copy(newQue, que.messages())

	q.lock.Lock()
	defer q.lock.Unlock()
	q.que = newQue
}

func (q *messageQue) len() int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return len(q.que)
}

func (q *messageQue) messages() []MessageTuple {
	q.lock.RLock()
	msgs := q.que
	q.lock.RUnlock()

	q.lock.Lock()
	defer q.lock.Unlock()
	q.que = []MessageTuple{}
	return msgs
}
