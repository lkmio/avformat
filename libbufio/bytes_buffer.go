package libbufio

import "fmt"

type BytesBuffer interface {
	Seek(count int) error

	SeekBack(count int) error

	Offset() int

	Reset(data []byte)

	// Data 返回剩余可读切片
	Data() []byte

	// ReadableBytes 返回剩余可读切片长度
	ReadableBytes() int
}

type bytesBuffer struct {
	data   []byte
	length int //data长度
	offset int //偏移量,从0开始
}

// 窥探剩余数据长度是否满足本次读取大小
func (b *bytesBuffer) peekN(count int) error {
	end := b.offset + count
	if b.length < end || end < 0 {
		return fmt.Errorf("slice bounds out of range [:%d] with length %d", b.offset+count, b.length)
	}

	b.offset = end
	return nil
}

func (b *bytesBuffer) setData(data []byte) {
	b.data = data
	b.offset = 0
	b.length = len(data)
}

func (b *bytesBuffer) Seek(count int) error {
	if err := b.peekN(count); err != nil {
		return err
	}

	return nil
}

func (b *bytesBuffer) SeekBack(count int) error {
	if err := b.peekN(0 - count); err != nil {
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

func (b *bytesBuffer) Data() []byte {
	return b.data[b.offset:]
}

func (b *bytesBuffer) ReadableBytes() int {
	return b.length - b.offset
}
