package event

import (
	"sync"
)

type Queue struct {
	items []Message
	mu    sync.Mutex
  cond *sync.Cond
}

func NewQueue() *Queue {
  q := &Queue{}
  q.cond = sync.NewCond(&q.mu)
  return q
}

func (q *Queue) Enqueue(item Message) {
  q.mu.Lock()
  defer q.mu.Unlock()
	q.items = append(q.items, item)
  q.cond.Signal()
}

func (q *Queue) Dequeue() Message {
  q.mu.Lock()
  defer q.mu.Unlock()
	for len(q.items) == 0 {
    return nil
	}
	msg := q.items[0]
	q.items = q.items[1:]
	return msg
}

func (q *Queue) Peek() Message {
  q.mu.Lock()
  defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	return q.items[0]
}

func (q *Queue) Size() int {
  q.mu.Lock()
  defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue) IsEmpty() bool {
  q.mu.Lock()
  defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cond.Broadcast()
}
