package libbufio

import (
	"encoding/binary"
)

type BytesReader interface {
	BytesBuffer

	ReadUint() (uint8, error)

	ReadUint16() (uint16, error)

	ReadUint24() (uint32, error)

	ReadUint32() (uint32, error)

	ReadUint64() (uint64, error)

	ReadBytes(count int) ([]byte, error)
}

func NewBytesReader(data []byte) BytesReader {
	b := &bytesReader{}
	b.setData(data)
	return b
}

// bytesReader 封装切片读操作
// 暂时不考虑使用接口封装, 避免栈上内存逃逸
type bytesReader struct {
	bytesBuffer
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
