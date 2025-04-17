package bufio

// ReadBits 从字节切片中读取指定索引和长度的bit值
func ReadBits(data []byte, startBit int, bitLength int) uint64 {
	var result uint64
	currentBit := startBit

	for i := 0; i < bitLength; i++ {
		byteIndex := currentBit / 8
		bitIndex := 7 - (currentBit % 8) // 因为字节中的bit顺序是从高位到低位

		if byteIndex >= len(data) {
			break // 超出数据范围
		}

		// 获取当前bit的值
		bitValue := (data[byteIndex] >> bitIndex) & 0x01
		result = (result << 1) | uint64(bitValue)

		currentBit++
	}

	return result
}

type BitsReader struct {
	Data   []byte
	Offset int
}

func (b *BitsReader) Read(length int) uint64 {
	v := ReadBits(b.Data, b.Offset, length)
	b.Offset += length
	return v
}

func (b *BitsReader) Seek(length int) {
	b.Offset += length
}

func (b *BitsReader) ReadBytes(length int) []byte {
	offset := b.Offset
	b.Offset += 8 * length
	return b.Data[offset/8 : b.Offset/8]
}

func (b *BitsReader) SafeRead(length int) (uint64, error) {
	return 0, nil
}
