package collections

import (
	"github.com/lkmio/avformat/utils"
)

type RingBuffer[T any] interface {
	IsEmpty() bool

	IsFull() bool

	Push(value T)

	Pop() T

	Head() T

	Tail() T

	Size() int

	Capacity() int

	Data() ([]T, []T)

	Clear()
}

type ringBuffer[T any] struct {
	data     []T
	head     int
	tail     int
	size     int
	capacity int
	zero     T
}

func (r *ringBuffer[T]) IsEmpty() bool {
	return r.size == 0
}

func (r *ringBuffer[T]) IsFull() bool {
	return r.size == cap(r.data)
}

func (r *ringBuffer[T]) Push(value T) {
	if r.IsFull() {
		r.Pop()
	}

	r.data[r.tail] = value
	r.tail = (r.tail + 1) % cap(r.data)

	r.size++
}

func (r *ringBuffer[T]) Pop() T {
	if r.IsEmpty() {
		return r.zero
	}

	element := r.data[r.head]
	r.data[r.head] = r.zero
	r.head = (r.head + 1) % cap(r.data)
	r.size--

	if r.size == 0 {
		r.Clear()
	}
	return element
}

func (r *ringBuffer[T]) Head() T {
	utils.Assert(!r.IsEmpty())
	return r.data[r.head]
}

func (r *ringBuffer[T]) Tail() T {
	utils.Assert(!r.IsEmpty())
	if r.tail > 0 {
		return r.data[r.tail-1]
	} else {
		return r.data[cap(r.data)-1]
	}
}

func (r *ringBuffer[T]) Size() int {
	return r.size
}

func (r *ringBuffer[T]) Capacity() int {
	return r.capacity
}

func (r *ringBuffer[T]) Data() ([]T, []T) {
	if r.size == 0 {
		return nil, nil
	}

	if r.tail <= r.head {
		return r.data[r.head:], r.data[:r.tail]
	} else {
		return r.data[r.head:r.tail], nil
	}
}

func (r *ringBuffer[T]) Clear() {
	r.size = 0
	r.head = 0
	r.tail = 0
}

func NewRingBuffer[T any](capacity int) RingBuffer[T] {
	utils.Assert(capacity > 0)
	r := &ringBuffer[T]{
		data:     make([]T, capacity),
		head:     0,
		tail:     0,
		size:     0,
		capacity: capacity,
	}

	return r
}
