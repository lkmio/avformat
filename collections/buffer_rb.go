package collections

import (
	"github.com/lkmio/avformat/bufio"
	"github.com/lkmio/avformat/utils"
)

// RBBlockBuffer 为AVPacket实现的环形缓冲区. 扩容时不重新拷贝旧的缓冲区(已经被AVPacket引用, 重新拷贝无意义)
type RBBlockBuffer struct {
	DirectBlockBuffer
	capacity          int // 有效容量, 不直接使用cap(r.data), 回环后尾部将有无法使用的内存空间
	discardBlockCount int // 扩容时, 丢弃之前的内存块数量
}

func (r *RBBlockBuffer) grow(capacity int) {
	r.data = make([]byte, capacity)
	r.capacity = capacity
}

func (r *RBBlockBuffer) Write(data []byte) {
	copy(r.Alloc(len(data)), data)
}

func (r *RBBlockBuffer) Alloc(size int) []byte {
	head, tail := 0, 0
	var oldBlock *struct {
		index     int
		size      int
		completed bool
	}

	if 0 == r.blocks.Size() {
		head = 0
		tail = 0
		r.capacity = cap(r.data)
	} else if 1 == r.blocks.Size() {
		oldBlock = r.blocks.Head()
		head = oldBlock.index
		tail = head + oldBlock.size
		r.capacity = cap(r.data)
	} else {
		oldBlock = r.blocks.Tail()
		head = r.blocks.Head().index
		tail = oldBlock.index + oldBlock.size
	}

	// 分配新的内存块
	if oldBlock == nil || oldBlock.completed {
		oldBlock = &struct {
			index     int
			size      int
			completed bool
		}{
			index:     tail,
			size:      0,
			completed: false,
		}

		r.blocks.Push(oldBlock)
	}

	over := tail < head
	if over && head-tail >= size {
		// 已经回环, 并且头部有大小合适的内存空间
	} else if !over && r.capacity-tail >= size {
		// 尾部有大小合适的内存空间
	} else if !over && head >= oldBlock.size+size {
		// 形成回环, 尾部空间不够, 但是头部空间足够

		// 修改有效内存容量大小, 前一个完整block的结束位置
		r.capacity = tail - oldBlock.size
		// 拷贝已写入的数据到头部
		copy(r.data, r.data[oldBlock.index:oldBlock.index+oldBlock.size])
		oldBlock.index = 0
		tail = oldBlock.size
	} else {
		// 扩容
		tmp := r.data
		r.grow((cap(r.data) + size) * 3 / 2)
		// 丢弃之前的数据, 只保留当前最后一个block
		copy(r.data, tmp[oldBlock.index:oldBlock.index+oldBlock.size])
		oldBlock.index = 0
		tail = oldBlock.size

		r.discardBlockCount = bufio.MaxInt(r.blocks.Size()-1, 0)
		for i := 0; i < r.discardBlockCount; i++ {
			r.blocks.Pop()
		}
	}

	bytes := r.data[tail : tail+size]
	r.blocks.Tail().size += size
	return bytes
}

func (r *RBBlockBuffer) Pop() {
	utils.Assert(!r.sealed)

	if r.discardBlockCount > 0 {
		r.discardBlockCount--
		return
	}

	r.DirectBlockBuffer.Pop()

	// 恢复容量
	if size := r.blocks.Size(); size == 0 || r.blocks.Head().index == 0 {
		r.capacity = cap(r.data)
	}
}

func (r *RBBlockBuffer) Clear() {
	r.DirectBlockBuffer.Clear()
	r.capacity = cap(r.data)
	r.discardBlockCount = 0
}

func (r *RBBlockBuffer) Data() ([]byte, []byte) {
	return nil, nil
}

func (r *RBBlockBuffer) AvailableBytes() int {
	//head, tail := 0, 0
	//if 0 == r.blocks.Size() {
	//	head = 0
	//	tail = 0
	//} else if 1 == r.blocks.Size() {
	//	oldBlock := r.blocks.Head()
	//	head = oldBlock.index
	//	tail = head + oldBlock.size
	//} else {
	//	oldBlock := r.blocks.Tail()
	//	head = r.blocks.Head().index
	//	tail = oldBlock.index + oldBlock.size
	//}
	return 0
}

func NewRBBlockBuffer(capacity int) *RBBlockBuffer {
	return &RBBlockBuffer{
		DirectBlockBuffer: DirectBlockBuffer{
			data: make([]byte, capacity),
			blocks: NewQueue[*struct {
				index     int
				size      int
				completed bool
			}](64),
		},

		capacity:          capacity,
		discardBlockCount: 0,
	}
}
