package bufio

import "fmt"

type BytesBuffer interface {
	// Seek 向右移动指定字节长度(相对位置)
	Seek(size int) error

	// SeekBack 向左移动指定字节长度(相对位置)
	SeekBack(size int) error

	// Offset 返回读写偏移量
	Offset() int

	Reset(data []byte)

	// RemainingBytes 返回剩余可读写的切片
	RemainingBytes() []byte

	// ReadableBytes 返回剩余可读写切片长度
	ReadableBytes() int

	// Clear 重置读写偏移量
	Clear()
}

type bytesBuffer struct {
	data   []byte
	length int // data长度
	offset int // 读写偏移量,从0开始
}

// 窥探剩余数据长度是否满足本次读取大小
func (b *bytesBuffer) peekN(size int) error {
	end := b.offset + size
	if b.length < end || end < 0 {
		return fmt.Errorf("slice bounds out of range [:%d] with length %d", b.offset+size, b.length)
	}

	b.offset = end
	return nil
}

func (b *bytesBuffer) setData(data []byte) {
	b.data = data
	b.offset = 0
	b.length = len(data)
}

func (b *bytesBuffer) Seek(size int) error {
	if err := b.peekN(size); err != nil {
		return err
	}

	return nil
}

func (b *bytesBuffer) SeekBack(size int) error {
	if err := b.peekN(0 - size); err != nil {
		return err
	}

	return nil
}

func (b *bytesBuffer) Offset() int {
	return b.offset
}

func (b *bytesBuffer) Reset(data []byte) {
	b.setData(data)
}

func (b *bytesBuffer) RemainingBytes() []byte {
	return b.data[b.offset:]
}

func (b *bytesBuffer) ReadableBytes() int {
	return b.length - b.offset
}

func (b *bytesBuffer) Clear() {
	b.offset = 0
}
