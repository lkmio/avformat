package libavc

import (
	"github.com/yangjiechina/avformat/utils"
)

var (
	StartCode3 = []byte{0x00, 0x00, 0x01}
	StartCode4 = []byte{0x00, 0x00, 0x00, 0x01}
)

func FindStartCode(p []byte, offset int) int {
	length := len(p)
	i := offset + 2

	for i < length {
		if p[i] > 1 {
			i += 3
		} else if p[i-1] != 0 {
			i += 2
		} else if (p[i-2] | (p[i] - 1)) != 0 {
			i++
		} else {
			i++
			break
		}
	}

	if i < length {
		return i
	} else {
		return -1
	}
}

func FindStartCodeFromBuffer(buffer utils.ByteBuffer, offset int) int {
	length := buffer.Size()
	i := offset + 2

	for i < length {
		if buffer.At(i) > 1 {
			i += 3
		} else if buffer.At(i-1) != 0 {
			i += 2
		} else if (buffer.At(i-2) | (buffer.At(i) - 1)) != 0 {
			i++
		} else {
			i++
			break
		}
	}

	if i < length {
		return i
	} else {
		return -1
	}
}

func IsKeyFrame(p []byte) bool {
	index := 0
	for {
		index = FindStartCode(p, index)
		if index < 0 {
			return false
		}
		state := p[index]
		switch state & 0x1F {
		case H264NalSPS:
			break
		case H264NalPPS:
			break
		case H264NalSEI:
			break
		case H264NalIDRSlice:
			return true
		case H264NalSlice:
			return false
		default:
			return false
		}
	}
}

func IsKeyFrameFromBuffer(buffer utils.ByteBuffer) bool {
	index := 0
	for {
		index = FindStartCodeFromBuffer(buffer, index)
		if index < 0 {
			return false
		}
		state := buffer.At(index)
		switch state & 0x1F {
		case H264NalSPS:
			break
		case H264NalPPS:
			break
		case H264NalSEI:
			break
		case H264NalIDRSlice:
			return true
		case H264NalSlice:
			return false
		default:
			return false
		}
	}
}

func ParseNalUnits(p []byte) int {
	for {
		index := FindStartCode(p, 0)
		state := p[index]
		switch state & 0x1F {
		case H264NalSlice:
		case H264NalIDRSlice:
			break
		}
	}
}

func copyNalU(buffer utils.ByteBuffer, data []byte, outSize int, append bool) int {
	var startCodeSize int

	if append {
		if outSize == 0 {
			startCodeSize = 4
		} else {
			startCodeSize = 3
		}

		if startCodeSize == 4 {
			buffer.Write(StartCode4)
		} else if startCodeSize != 0 {
			buffer.Write(StartCode3)
		}
	}

	buffer.Write(data)

	return startCodeSize + len(data)
}
