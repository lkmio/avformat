package collections

import "github.com/lkmio/avformat/utils"

// DirectBlockBuffer 水平内存池, 适合用于合并写块, 只写入和读取, 不删除.
type DirectBlockBuffer struct {
	data       []byte
	nextOffset int  // 下一个block的偏移量
	sealed     bool // 是否已经封存, 不可再写入

	blocks *Queue[*struct {
		index     int // 位于data中的偏移量
		size      int
		completed bool
	}]
}

func (b *DirectBlockBuffer) grow(capacity int) {
	bytes := make([]byte, capacity)
	copy(bytes, b.data[:b.nextOffset])
	b.data = bytes
}

func (b *DirectBlockBuffer) Write(data []byte) {
	copy(b.Alloc(len(data)), data)
}

func (b *DirectBlockBuffer) Alloc(size int) []byte {
	utils.Assert(!b.sealed)

	var block *struct {
		index     int
		size      int
		completed bool
	}

	if !b.blocks.IsEmpty() {
		block = b.blocks.Tail()
	}

	if block == nil || block.completed {
		// 分配新的内存块
		block = &struct {
			index     int
			size      int
			completed bool
		}{
			index:     b.nextOffset,
			size:      0,
			completed: false,
		}

		b.blocks.Push(block)
	}

	// 扩容
	nextOffset := b.nextOffset + size
	if nextOffset > cap(b.data) {
		capacity := nextOffset * 3 / 2
		b.grow(capacity)
	}

	block.size += size
	data := b.data[b.nextOffset:nextOffset]
	b.nextOffset = nextOffset
	return data
}

func (b *DirectBlockBuffer) Feat() []byte {
	utils.Assert(!b.sealed)
	block := b.blocks.Tail()
	block.completed = true
	return b.data[block.index : block.index+block.size]
}

func (b *DirectBlockBuffer) Pop() {
	utils.Assert(!b.sealed)

	b.blocks.Pop()
	if b.blocks.IsEmpty() {
		b.nextOffset = 0
	}
}

func (b *DirectBlockBuffer) PopBack() {
	utils.Assert(!b.sealed)

	b.blocks.PopBack()
	if b.blocks.IsEmpty() {
		b.nextOffset = 0
	}
}

func (b *DirectBlockBuffer) PendingBlockSize() int {
	if !b.blocks.IsEmpty() && !b.blocks.Tail().completed {
		return b.blocks.Tail().size
	}

	return 0
}

func (b *DirectBlockBuffer) Clear() {
	b.sealed = false
	b.blocks.Clear()
	b.nextOffset = 0
}

func (b *DirectBlockBuffer) Data() []byte {
	return b.data[:b.nextOffset]
}

func (b *DirectBlockBuffer) AvailableBytes() int {
	return cap(b.data) - b.nextOffset
}

func (b *DirectBlockBuffer) Size() int {
	return b.blocks.Size()
}

func (b *DirectBlockBuffer) SplitOff() BlockBuffer {
	b.sealed = true
	return &DirectBlockBuffer{
		data: b.data[b.nextOffset:],
		blocks: NewQueue[*struct {
			index     int
			size      int
			completed bool
		}](64),
		nextOffset: b.nextOffset,
	}
}

func NewDirectBlockBuffer(capacity int) BlockBuffer {
	return &DirectBlockBuffer{
		data: make([]byte, capacity),
		blocks: NewQueue[*struct {
			index     int
			size      int
			completed bool
		}](64),
	}
}
