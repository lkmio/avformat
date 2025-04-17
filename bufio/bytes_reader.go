package bufio

import (
	"encoding/binary"
)

type BytesReader interface {
	BytesBuffer

	ReadUint8() (uint8, error)

	ReadUint16() (uint16, error)

	ReadUint24() (uint32, error)

	ReadUint32() (uint32, error)

	ReadUint64() (uint64, error)

	ReadBytes(size int) ([]byte, error)
}

// bytesReader 封装切片读操作
type bytesReader struct {
	bytesBuffer
}

func (b *bytesReader) ReadUint8() (uint8, error) {
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

	return Uint24(b.data[b.offset-3:]), nil
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

func (b *bytesReader) ReadBytes(size int) ([]byte, error) {
	tmp := b.offset
	if err := b.peekN(size); err != nil {
		return nil, err
	}

	return b.data[tmp:b.offset], nil
}

func NewBytesReader(data []byte) BytesReader {
	b := &bytesReader{}
	b.setData(data)
	return b
}
