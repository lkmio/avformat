package bufio

// WriteBits 将指定的bit值写入字节切片的指定位置
func WriteBits(data []byte, startBit int, bitLength int, value uint64) {
	currentBit := startBit

	for i := bitLength - 1; i >= 0; i-- {
		byteIndex := currentBit / 8
		bitIndex := 7 - (currentBit % 8) // 因为字节中的bit顺序是从高位到低位

		if byteIndex >= len(data) {
			break // 超出数据范围
		}

		// 获取当前bit的值
		bitValue := (value >> uint(i)) & 0x01

		// 将bit值写入指定位置
		data[byteIndex] &^= (1 << bitIndex)           // 先将目标bit清零
		data[byteIndex] |= byte(bitValue << bitIndex) // 然后设置bit值

		currentBit++
	}
}

type BitsWriter struct {
	Data   []byte
	Offset int
}

func (b *BitsWriter) Write(length int, value uint64) {
	WriteBits(b.Data, b.Offset, length, value)
	b.Offset += length
}

func (b *BitsWriter) Seek(length int) {
	b.Offset += length
}

func (b *BitsWriter) WriteBytes(data []byte) {
	copy(b.Data[b.Offset/8:], data)
	b.Offset += len(data) * 8
}
