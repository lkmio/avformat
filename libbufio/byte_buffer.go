package libbufio

import "fmt"

// ByteBuffer unsafe thread
type ByteBuffer interface {
	Write(data []byte)
	Size() int
	Clear()
	ReadInt16() int16
	ReadInt24() int32
	ReadInt32() int32
	ReadInt64() int64
	ReadUInt8() uint8
	ReadUInt16() uint16
	ReadUInt24() uint32
	ReadUInt32() uint32
	ReadUInt64() uint64
	ReadTo(func([]byte))
	ReadBytes(dst []byte) int
	ToBytes() []byte
	ReadOffset() int
	ReadableBytes() int
	ReadBytesWithShallowCopy(count int) []byte

	PeekCount(int) error
	PeekUInt8() uint8
	PeekUInt16() uint16
	PeekUInt24() uint32
	PeekUInt32() uint32
	PeekUInt64() uint64
	PeekTo(func([]byte))

	Skip(count int)
	At(index int) byte
}

type byteBuffer struct {
	data       [][]byte
	itemSize   []int
	size       int
	readOffset int
}

func (b *byteBuffer) PeekCount(i int) error {
	if bytes := b.ReadableBytes(); i > bytes {
		return fmt.Errorf(" runtime error: slice bounds out of range [:%d] with capacity %d", i, bytes)
	}
	return nil
}

func NewByteBuffer(data ...[]byte) ByteBuffer {
	buffer := &byteBuffer{}
	for _, datum := range data {
		buffer.Write(datum)
	}
	return buffer
}

func (b *byteBuffer) Write(data []byte) {
	b.data = append(b.data, data)
	b.size += len(data)
	b.itemSize = append(b.itemSize, b.size)
}

func (b *byteBuffer) Size() int {
	return b.size
}

func (b *byteBuffer) Clear() {
	b.data = nil
	b.size = 0
	b.itemSize = nil
	b.readOffset = 0
}

func (b *byteBuffer) ToBytes() []byte {

	dst := make([]byte, b.size-b.readOffset)
	offset := 0
	b.PeekTo(func(bytes []byte) {
		copy(dst[offset:], bytes)
		offset += len(bytes)
		b.readOffset = offset
	})

	return dst
}

func (b *byteBuffer) PeekTo(handle func([]byte)) {
	i1, i2 := b.offset()
	for i, bytes := range b.data[i1:] {
		if i == 0 {
			handle(bytes[i2:])
		} else {
			handle(bytes)
		}
	}
}

func (b *byteBuffer) ReadTo(handle func([]byte)) {
	b.PeekTo(handle)
	b.readOffset = b.size
}

// 返回readOffset在二维切片的索引
func (b *byteBuffer) offset() (int, int) {
	if len(b.itemSize) == 1 {
		return 0, b.readOffset
	}

	for i, v := range b.itemSize {
		if b.readOffset < v {
			if i > 0 {
				return i, b.readOffset - b.itemSize[i-1]
			} else {
				return 0, b.readOffset
			}
		}
	}

	panic("slice index out of range")
}

func (b *byteBuffer) At(index int) byte {
	if len(b.itemSize) == 1 {
		return b.data[0][index]
	}

	for i, v := range b.itemSize {
		if index < v {
			if i > 0 {
				return b.data[i][index-b.itemSize[i-1]]
			} else {
				return b.data[0][index]
			}
		}

	}

	panic("slice index out of range")
}

//func (b *byteBuffer) ForEach(start int, handle func(i int, v byte) (bool, int)) {
//	index := 0
//	offset := start
//	if start >= b.size {
//		panic("slice index out of range")
//	}
//
//	for i := 0; i < len(b.data); i++ {
//		bytes := b.data[i]
//		length := len(bytes)
//		total := index + length
//
//		if offset > length {
//			offset -= length
//			index = total
//			continue
//		}
//		for j := offset; j < length; {
//			if broken, next := handle(index, bytes[i]); broken {
//				return
//			} else {
//				if next >= b.size {
//					panic("slice index out of range")
//				}
//				if next < total {
//					j = next
//				} else {
//					offset = length - j
//				}
//			}
//		}
//
//		index = total
//	}
//}

func (b *byteBuffer) ReadInt16() int16 {
	return int16(b.ReadUInt16())
}

func (b *byteBuffer) ReadInt24() int32 {
	return int32(b.ReadUInt24())
}

func (b *byteBuffer) ReadInt32() int32 {
	return int32(b.ReadUInt32())
}

func (b *byteBuffer) ReadInt64() int64 {
	return int64(b.ReadUInt64())
}

func (b *byteBuffer) ReadUInt8() uint8 {
	i := b.PeekUInt8()
	b.readOffset++
	return i
}

func (b *byteBuffer) ReadUInt16() uint16 {
	i := b.PeekUInt16()
	b.readOffset += 2
	return i
}

func (b *byteBuffer) ReadUInt24() uint32 {
	i := b.PeekUInt24()
	b.readOffset += 3
	return i
}

func (b *byteBuffer) ReadUInt32() uint32 {
	i := b.PeekUInt32()
	b.readOffset += 4
	return i
}

func (b *byteBuffer) ReadUInt64() uint64 {
	i := b.PeekUInt64()
	b.readOffset += 8
	return i
}

func (b *byteBuffer) PeekUInt8() uint8 {
	return b.At(b.readOffset)
}

func (b *byteBuffer) PeekUInt16() uint16 {
	return BytesToUInt16(b.At(b.readOffset), b.At(b.readOffset+1))
}

func (b *byteBuffer) PeekUInt24() uint32 {
	return UInt24(b.At(b.readOffset), b.At(b.readOffset+1), b.At(b.readOffset+2))
}

func (b *byteBuffer) PeekUInt32() uint32 {
	return BytesToUInt32(b.At(b.readOffset), b.At(b.readOffset+1), b.At(b.readOffset+2), b.At(b.readOffset+3))
}

func (b *byteBuffer) PeekUInt64() uint64 {
	return BytesToUInt64(b.At(b.readOffset), b.At(b.readOffset+1), b.At(b.readOffset+2), b.At(b.readOffset+3), b.At(b.readOffset+4), b.At(b.readOffset+5), b.At(b.readOffset+6), b.At(b.readOffset+7))
}

func (b *byteBuffer) Skip(count int) {
	b.readOffset += count
	if b.readOffset > b.size {
		panic("slice index out of range")
	}
}

func (b *byteBuffer) ReadOffset() int {
	return b.readOffset
}

func (b *byteBuffer) ReadableBytes() int {
	return b.size - b.readOffset
}

func (b *byteBuffer) ReadBytes(dst []byte) int {
	dstSize := MinInt(b.ReadableBytes(), len(dst))
	line, column := b.offset()
	index := 0

	for i := line; i < len(b.data) && index < dstSize; i++ {
		bytes := b.data[i]
		if i == line {
			bytes = bytes[column:]
		}

		size := MinInt(dstSize-index, len(bytes))
		copy(dst[index:], bytes[:size])
		index += size
		b.readOffset += size
	}

	return dstSize
}

// ReadBytesWithShallowCopy can only be used when the data length is 1.
func (b *byteBuffer) ReadBytesWithShallowCopy(count int) []byte {
	end := b.readOffset + count
	bytes := b.data[0][b.readOffset:end]
	b.readOffset = end
	return bytes
}
