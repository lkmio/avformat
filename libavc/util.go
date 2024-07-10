package libavc

import (
	"encoding/binary"
	"fmt"
	"github.com/lkmio/avformat/libbufio"
)

var (
	StartCode3 = []byte{0x00, 0x00, 0x01}
	StartCode4 = []byte{0x00, 0x00, 0x00, 0x01}
)

// FindStartCode 返回NalUHeader位置
func FindStartCode(p []byte) int {
	length := len(p)
	i := 2

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

// FindStartCodeWithReader 返回start code值
func FindStartCodeWithReader(reader libbufio.BytesReader) int {
	data := reader.Data()
	index := FindStartCode(data)
	if index < 0 {
		return -1
	}

	_ = reader.Seek(index + 1)
	return int(data[index])
}

// FindStartCode2 返回start code的起始位置
func FindStartCode2(p []byte) int {
	index := FindStartCode(p)
	if index < 0 {
		return index
	}

	index -= 4

	if index < 0 {
		return 0
	} else if p[index] == 0 {
		return index
	} else {
		return index + 1
	}
}

func IsKeyFrame(p []byte) bool {
	index := 0
	for {
		index = FindStartCode(p[index:])
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

func ParseNalUnits(p []byte) int {
	for {
		index := FindStartCode(p)
		state := p[index]
		switch state & 0x1F {
		case H264NalSlice:
		case H264NalIDRSlice:
			break
		}
	}
}

func copyNalUWithBytes(dst []byte, data []byte, outSize int, append bool) int {
	var startCodeSize int
	var offset int

	if append {
		if outSize == 0 {
			startCodeSize = 4
		} else {
			startCodeSize = 3
		}

		if startCodeSize == 4 {
			binary.BigEndian.PutUint32(dst[offset:], 0x1)
			offset += 4
		} else if startCodeSize != 0 {
			libbufio.WriteUInt24(dst[offset:], 0x1)
			offset += 3
		}
	}

	copy(dst[offset:], data)
	return startCodeSize + len(data)
}

func copyNalU(buffer libbufio.ByteBuffer, data []byte, outSize int, append bool) int {
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

func AVCC2AnnexB(dst []byte, avcc []byte, extra []byte) int {
	length := len(avcc)
	outSize, spsSeen, ppsSeen := 0, false, false

	for index := 4; index < length; index += 4 {
		size := int(binary.BigEndian.Uint32(avcc[index-4:]))
		if size == 0 || length-index < size {
			return outSize
		}

		unitType := avcc[index] & 0x1F
		switch unitType {
		case H264NalSPS:
			spsSeen = true
			break
		case H264NalPPS:
			ppsSeen = true
			break
		case H264NalIDRSlice:
			if !spsSeen && !ppsSeen && len(extra) > 0 {
				outSize += copyNalUWithBytes(dst[outSize:], extra, outSize, false)
			}
			break
		case H264NalSEI:
			index += size
			continue
		}

		bytes := avcc[index : index+size]
		outSize += copyNalUWithBytes(dst[outSize:], bytes, outSize, true)
		index += size
	}

	return outSize
}

func AnnexB2AVCC(dst []byte, annexB []byte) int {
	length := len(annexB)
	size := 0
	nalStart := FindStartCode(annexB)
	if nalStart < 0 {
		return 0
	}

	for {
		for nalStart < length && annexB[nalStart] == 0 {
			nalStart++
		}

		if nalStart == length {
			return size
		}

		nalEnd := FindStartCode(annexB[nalStart:])
		if nalEnd < 0 {
			nalEnd = len(annexB[nalStart:])
		}

		nalSize := nalEnd - 4
		if nalSize < 0 || annexB[nalStart:][nalSize] != 0x0 {
			nalSize++
		}

		binary.BigEndian.PutUint32(dst[size:], uint32(nalSize))
		size += 4

		copy(dst[size:], annexB[nalStart:nalStart+nalSize])
		size += nalSize
		nalStart += nalEnd
	}
}

func Mp4ToAnnexB(buffer libbufio.ByteBuffer, data, extra []byte) {
	length := len(data)
	outSize, spsSeen, ppsSeen := 0, false, false
	for index := 4; index < length; index += 4 {
		size := int(binary.BigEndian.Uint32(data[index-4:]))
		if size == 0 || length-index < size {
			break
		}
		unitType := data[index] & 0x1F
		switch unitType {
		case H264NalSPS:
			spsSeen = true
			break
		case H264NalPPS:
			ppsSeen = true
			break
		case H264NalIDRSlice:
			if !spsSeen && !ppsSeen {
				outSize += copyNalU(buffer, extra, outSize, false)
			}
			break
		}

		bytes := data[index : index+size]
		outSize += copyNalU(buffer, bytes, outSize, true)
		index += size
	}
}

func M4VCExtraDataToAnnexB(src []byte) ([]byte, error) {
	buffer := libbufio.NewByteBuffer(src)
	//unsigned int(8) configurationVersion = 1;
	//unsigned int(8) AVCProfileIndication;
	//unsigned int(8) profile_compatibility;
	//unsigned int(8) AVCLevelIndication;
	if err := buffer.PeekCount(6); err != nil {
		return nil, err
	}

	buffer.Skip(4)
	_ = buffer.ReadUInt8()&0x3 + 1
	unitNb := buffer.ReadUInt8() & 0x1f
	dstBuffer := libbufio.NewByteBuffer()
	spsDone := 0
	for unitNb != 0 {
		unitNb--

		if err := buffer.PeekCount(2); err != nil {
			return nil, err
		}
		size := int(buffer.ReadUInt16())
		dstBuffer.Write(StartCode4)
		readOffset := buffer.ReadOffset()
		dstBuffer.Write(src[readOffset : readOffset+size])
		buffer.Skip(size)

		spsDone++
		if buffer.ReadableBytes() > 2 && unitNb == 0 && spsDone == 1 {
			unitNb = buffer.ReadUInt8()
		}
	}

	return dstBuffer.ToBytes(), nil
}

func SplitNalU(data []byte, cb func(nalu []byte)) {
	var offset int
	//+3查找第二个nalu
	for n := FindStartCode2(data[offset+3:]); n > -1; n = FindStartCode2(data[offset+3:]) {
		n += 3
		cb(data[offset : offset+n])
		offset += n
	}

	cb(data[offset:])
}

func RemoveStartCode(data []byte) []byte {
	if data[0] != 0 || data[1] != 0 {
		return data
	} else if data[2] == 0x1 {
		return data[3:]
	} else if data[2] == 0x0 && data[3] == 0x1 {
		return data[4:]
	}

	return data
}

// ParseExtraDataFromKeyNALU 从关键帧中解析出sps/pss
func ParseExtraDataFromKeyNALU(data []byte) ([]byte, []byte, error) {
	var sps []byte
	var pps []byte

	SplitNalU(data, func(nalu []byte) {
		noStartCodeNALU := RemoveStartCode(nalu)
		header := noStartCodeNALU[0] & 0x1F

		if byte(H264NalSPS) == header {
			sps = make([]byte, len(noStartCodeNALU))
			copy(sps, noStartCodeNALU)
		} else if byte(H264NalPPS) == header {
			pps = make([]byte, len(noStartCodeNALU))
			copy(pps, noStartCodeNALU)
		}
	})

	if sps == nil || pps == nil {
		return nil, nil, fmt.Errorf("not find extra data for H264")
	}
	return sps, pps, nil
}
