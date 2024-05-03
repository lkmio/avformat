package libbufio

import (
	"encoding/binary"
	"fmt"
)

type BytesReader interface {
	ReadUint() (uint8, error)

	ReadUint16() (uint16, error)

	ReadUint24() (uint32, error)

	ReadUint32() (uint32, error)

	ReadUint64() (uint64, error)

	ReadBytes(count int) ([]byte, error)

	Seek(count int) error

	SeekBack(count int) error

	Offset() int

	Reset(data []byte)

	// Data 返回剩余可读切片
	Data() []byte

	// ReadableBytes 返回剩余可读切片长度
	ReadableBytes() int
}

func NewByteReader(data []byte) BytesReader {
	b := &bytesReader{}
	b.setData(data)
	return b
}

// bytesReader 封装切片读操作
// 暂时不考虑使用接口封装, 避免栈上内存逃逸
type bytesReader struct {
	data []byte

	length int
	offset int
}

// 窥探剩余数据长度是否满足本次读取大小
func (b *bytesReader) peekN(count int) error {
	end := b.offset + count
	if b.length < end || end < 0 {
		return fmt.Errorf("slice bounds out of range [:%d] with length %d", b.offset+count, b.length)
	}

	b.offset = end
	return nil
}

func (b *bytesReader) ReadUint() (uint8, error) {
	if err := b.peekN(1); err != nil {
		return 0, err
	}

	return b.data[b.offset-1], nil
}

func (b *bytesReader) ReadUint16() (uint16, error) {
	if err := b.peekN(2); err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b.data[b.offset-2:]), nil
}

func (b *bytesReader) ReadUint24() (uint32, error) {
	if err := b.peekN(3); err != nil {
		return 0, err
	}

	return BytesToUInt24(b.data[b.offset-3:]), nil
}

func (b *bytesReader) ReadUint32() (uint32, error) {
	if err := b.peekN(4); err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b.data[b.offset-4:]), nil
}

func (b *bytesReader) ReadUint64() (uint64, error) {
	if err := b.peekN(8); err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(b.data[b.offset-8:]), nil
}

func (b *bytesReader) ReadBytes(count int) ([]byte, error) {
	tmp := b.offset
	if err := b.peekN(count); err != nil {
		return nil, err
	}

	return b.data[tmp:b.offset], nil
}

func (b *bytesReader) Seek(count int) error {
	if err := b.peekN(count); err != nil {
		return err
	}

	return nil
}

func (b *bytesReader) SeekBack(count int) error {
	if err := b.peekN(0 - count); err != nil {
		return err
	}

	return nil
}

func (b *bytesReader) Offset() int {
	return b.offset
}

func (b *bytesReader) setData(data []byte) {
	b.data = data
	b.offset = 0
	b.length = len(data)
}

func (b *bytesReader) Reset(data []byte) {
	b.setData(data)
}

func (b *bytesReader) Data() []byte {
	return b.data[b.offset:]
}

func (b *bytesReader) ReadableBytes() int {
	return b.length - b.offset
}
