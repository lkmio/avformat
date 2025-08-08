package collections

type BlockBuffer interface {
	// Write 从尾部写入指定长度
	Write(data []byte)

	// Alloc 从尾部分配指定内存长度
	Alloc(size int) []byte

	// Fetch 从尾部获取一个block
	Fetch() []byte

	// Pop 从头部弹出一个block
	Pop()

	// AvailableBytes 可用内存长度
	AvailableBytes() int

	// PopBack 从尾部弹出一个block
	PopBack()

	PendingBlockSize() int

	// SplitOff 分割出新的BlockBuffer, 新的BlockBuffer持有空闲内存块, 当前BlockBuffer将不再允许写入
	SplitOff() BlockBuffer

	Clear()

	Size() int
}
