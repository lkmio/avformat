package collections

import "sync/atomic"

type ReferenceCounter[T any] struct {
	refer int32
	data  T
}

func (r *ReferenceCounter[T]) Refer() {
	atomic.AddInt32(&r.refer, 1)
}

func (r *ReferenceCounter[T]) Release() bool {
	return atomic.AddInt32(&r.refer, -1) == 0
}

func (r *ReferenceCounter[T]) UseCount() int32 {
	return atomic.LoadInt32(&r.refer)
}

func (r *ReferenceCounter[T]) Get() T {
	return r.data
}

//func (r *ReferenceCounter[T]) Reset(data T) {
//	r.data = data
//	atomic.StoreInt32(&r.refer, 1)
//}

func (r *ReferenceCounter[T]) ResetData(data T) {
	r.data = data
}

func NewReferenceCounter[T any](data T) *ReferenceCounter[T] {
	return &ReferenceCounter[T]{
		refer: 1,
		data:  data,
	}
}
