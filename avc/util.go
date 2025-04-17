package avc

import (
	"encoding/binary"
	"fmt"
	"github.com/lkmio/avformat/bufio"
)

var (
	StartCode3 = []byte{0x00, 0x00, 0x01}
	StartCode4 = []byte{0x00, 0x00, 0x00, 0x01}
)

// FindStartCode 返回NalUHeader索引和start code长度
func FindStartCode(p []byte) (int, int) {
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
		// 计算start code长度
		size := 3
		if i > 3 && p[i-4] == 0x0 {
			size++
		}

		return i, size
	} else {
		return -1, -1
	}
}

// FindStartCodeWithReader 返回NalUHeader的值
func FindStartCodeWithReader(reader bufio.BytesReader) int {
	data := reader.RemainingBytes()
	index, _ := FindStartCode(data)
	if index < 0 {
		return -1
	}

	_ = reader.Seek(index + 1)
	return int(data[index])
}

// FindStartCode2 返回start code的起始位置
func FindStartCode2(p []byte) int {
	index, length := FindStartCode(p)
	if index < 0 {
		return index
	}

	return index - length
}

func IsKeyFrame(p []byte) bool {
	index := 0
	for {
		n, _ := FindStartCode(p[index:])
		if n < 0 {
			return false
		}

		index += n
		state := p[index]
		switch state & 0x1F {
		case H264NalSEI, H264NalAUD, H264NalPPS, H264NalSPS:
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
		index, _ := FindStartCode(p)
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
			bufio.PutUint24(dst[offset:], 0x1)
			offset += 3
		}
	}

	copy(dst[offset:], data)
	return startCodeSize + len(data)
}

func copyNalU(writer bufio.BytesWriter, data []byte, outSize int, append bool) int {
	var startCodeSize int

	if append {
		if outSize == 0 {
			startCodeSize = 4
		} else {
			startCodeSize = 3
		}

		if startCodeSize == 4 {
			_ = writer.Write(StartCode4)
		} else if startCodeSize != 0 {
			_ = writer.Write(StartCode3)
		}
	}

	_ = writer.Write(data)
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
	// avcc包大小
	size := 0
	nalStart, _ := FindStartCode(annexB)
	if nalStart < 0 {
		return 0
	}

	for nalStart < length {
		// 跳过开头空包(连续0000000100000001)
		for nalStart < length && annexB[nalStart] == 0 {
			nalStart++
		}
		if nalStart >= length {
			break
		}

		nalEnd, n := FindStartCode(annexB[nalStart:])
		// 最后一个nalu
		if nalEnd < 0 {
			nalEnd = len(annexB[nalStart:])
			n = 0
		}

		nalSize := nalEnd - n
		// 空包, 保存前一个nalu(1个字节header)
		if nalSize == 0 {
			nalSize++
		}

		binary.BigEndian.PutUint32(dst[size:], uint32(nalSize))
		size += 4

		copy(dst[size:], annexB[nalStart:nalStart+nalSize])
		size += nalSize
		nalStart += nalEnd
	}

	return size
}

func Mp4ToAnnexB(writer bufio.BytesWriter, data, extra []byte) {
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
				outSize += copyNalU(writer, extra, outSize, false)
			}
			break
		}

		bytes := data[index : index+size]
		outSize += copyNalU(writer, bytes, outSize, true)
		index += size
	}
}

func M4VCExtraDataToAnnexB(src []byte) ([]byte, error) {
	reader := bufio.NewBytesReader(src)

	//unsigned int(8) configurationVersion = 1;
	//unsigned int(8) AVCProfileIndication;
	//unsigned int(8) profile_compatibility;
	//unsigned int(8) AVCLevelIndication;
	if err := reader.Seek(4); err != nil {
		return nil, err
	}

	readUint8, err := reader.ReadUint8()
	if err != nil {
		return nil, err
	}

	readUint8 &= 0x3 + 1
	unitNb, err := reader.ReadUint8()
	unitNb &= 0x1f

	spsDone := 0
	writer := bufio.NewBytesWriter(make([]byte, len(src)+256))
	for unitNb != 0 {
		unitNb--

		var size uint16
		size, err = reader.ReadUint16()
		if err != nil {
			return nil, err
		} else if err = writer.Write(StartCode4); err != nil {
			return nil, err
		} else if err = writer.Write(src[reader.Offset() : reader.Offset()+int(size)]); err != nil {
			return nil, err
		} else if err = reader.Seek(int(size)); err != nil {
			return nil, err
		}

		spsDone++
		if reader.ReadableBytes() > 2 && unitNb == 0 && spsDone == 1 {
			unitNb, _ = reader.ReadUint8()
		}
	}

	return writer.WrittenBytes(), nil
}

func SplitNalU(data []byte, cb func(nalu []byte)) {
	var offset int
	// +3查找第二个nalu
	for n := FindStartCode2(data[offset+3:]); n > -1; n = FindStartCode2(data[offset+3:]) {
		n += 3
		cb(data[offset : offset+n])
		offset += n
	}

	// 回调最后一个nalu
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
			sps = make([]byte, 4+len(noStartCodeNALU))
			binary.BigEndian.PutUint32(sps, 0x1)
			copy(sps[4:], noStartCodeNALU)
		} else if byte(H264NalPPS) == header {
			pps = make([]byte, 4+len(noStartCodeNALU))
			binary.BigEndian.PutUint32(pps, 0x1)
			copy(pps[4:], noStartCodeNALU)
		}
	})

	if sps == nil || pps == nil {
		return nil, nil, fmt.Errorf("not find extra data for H264")
	}
	return sps, pps, nil
}
