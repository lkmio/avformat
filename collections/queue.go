package collections

import (
	"github.com/lkmio/avformat/utils"
)

type Queue[T any] struct {
	*ringBuffer[T]
}

func (q *Queue[T]) Push(value T) {
	if q.ringBuffer.IsFull() {
		newArray := make([]T, q.ringBuffer.Size()*2)
		head, tail := q.ringBuffer.Data()
		copy(newArray, head)
		if tail != nil {
			copy(newArray[len(head):], tail)
		}

		q.data = newArray
		q.head = 0
		q.tail = q.size
	}

	q.data[q.tail] = value
	q.tail = (q.tail + 1) % cap(q.data)

	q.size++
}

func (q *Queue[T]) PopBack() T {
	utils.Assert(q.size > 0)

	value := q.ringBuffer.Tail()
	q.size--
	q.tail = (q.tail - 1 + cap(q.data)) % cap(q.data)

	if q.size == 0 {
		q.Clear()
	}
	return value
}

func (q *Queue[T]) Peek(index int) T {
	head, tail := q.ringBuffer.Data()
	if index < len(head) {
		return head[index]
	} else {
		return tail[index-len(head)]
	}
}

func NewQueue[T any](capacity int) *Queue[T] {
	utils.Assert(capacity > 0)

	return &Queue[T]{ringBuffer: &ringBuffer[T]{
		data: make([]T, capacity),
		head: 0,
		tail: 0,
		size: 0,
	}}
}
