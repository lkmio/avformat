package utils

/**
全都大端子序
*/

func WriteWORD(dst []byte, src uint16) {
	dst[0] = byte(src >> 8)
	dst[1] = byte(src)
}

func WriteDWORD(dst []byte, src uint32) {
	dst[0] = byte(src >> 24)
	dst[1] = byte(src >> 16)
	dst[2] = byte(src >> 8)
	dst[3] = byte(src)
}

func WriteUInt24(dst []byte, src uint32) {
	dst[0] = byte(src >> 16)
	dst[1] = byte(src >> 8)
	dst[2] = byte(src)
}

func BytesToInt(src []byte) int {
	result := 0
	for i := 0; i < len(src); i++ {
		result <<= 8
		result |= int(src[i])
	}

	return result
}

func BytesToUInt16(b1, b2 byte) uint16 {
	return uint16(b1)<<8 | uint16(b2)
}

func BytesToUInt24(b1, b2, b3 byte) uint32 {
	return (uint32(b1) << 16) | (uint32(b2) << 8) | uint32(b3)
}

func BytesToUInt24WhitSlice(data []byte) uint32 {
	return (uint32(data[0]) << 16) | (uint32(data[1]) << 8) | uint32(data[2])
}

func BytesToUInt32(b1, b2, b3, b4 byte) uint32 {
	return (uint32(b1) << 24) | (uint32(b2) << 16) | (uint32(b3) << 8) | uint32(b4)
}

func BytesToUInt64(b1, b2, b3, b4, b5 byte, b6 byte, b7 byte, b8 byte) uint64 {
	return (uint64(b1) << 56) | (uint64(b2) << 48) | (uint64(b3) << 40) | (uint64(b4) << 32) | (uint64(b5) << 24) | (uint64(b6) << 16) | (uint64(b7) << 8) | uint64(b8)
}

func MinInt(a int, b int) int {
	if a > b {
		return b
	}

	return a
}

func ReadBits() {

}

func WriteBits() {

}
