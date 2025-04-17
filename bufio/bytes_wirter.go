package bufio

import (
	"encoding/binary"
)

type BytesWriter interface {
	BytesBuffer

	WriteUint8(data byte) error

	WriteUint16(data uint16) error

	WriteUint32(data uint32) error

	WriteUint64(data uint64) error

	Write(data []byte) error

	// WrittenBytes 返回已经写入的切片
	WrittenBytes() []byte
}

type bytesWriter struct {
	bytesBuffer
}

func (b *bytesWriter) WriteUint8(data byte) error {
	if err := b.peekN(1); err != nil {
		return err
	}

	b.data[b.offset-1] = data
	return nil
}

func (b *bytesWriter) WriteUint16(data uint16) error {
	if err := b.peekN(2); err != nil {
		return err
	}

	binary.BigEndian.PutUint16(b.data[b.offset-2:], data)
	return nil
}

func (b *bytesWriter) WriteUint32(data uint32) error {
	if err := b.peekN(4); err != nil {
		return err
	}

	binary.BigEndian.PutUint32(b.data[b.offset-4:], data)
	return nil
}

func (b *bytesWriter) WriteUint64(data uint64) error {
	if err := b.peekN(8); err != nil {
		return err
	}

	binary.BigEndian.PutUint64(b.data[b.offset-8:], data)
	return nil
}

func (b *bytesWriter) Write(data []byte) error {
	if err := b.peekN(len(data)); err != nil {
		return err
	}

	copy(b.data[b.offset-len(data):], data)
	return nil
}

func (b *bytesWriter) WrittenBytes() []byte {
	return b.data[:b.offset]
}

func NewBytesWriter(data []byte) BytesWriter {
	writer := bytesWriter{}
	writer.setData(data)
	return &writer
}
