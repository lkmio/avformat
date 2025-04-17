package bufio

func PutUint24(dst []byte, src uint32) {
	dst[0] = byte(src >> 16)
	dst[1] = byte(src >> 8)
	dst[2] = byte(src)
}

func Uint24(data []byte) uint32 {
	return (uint32(data[0]) << 16) | (uint32(data[1]) << 8) | uint32(data[2])
}

func MinInt(a int, b int) int {
	if a > b {
		return b
	}

	return a
}

func MaxInt(a int, b int) int {
	if a > b {
		return a
	}

	return b
}
