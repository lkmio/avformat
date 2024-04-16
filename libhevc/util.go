package libhevc

import (
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
)

var (
	StartCode3 = []byte{0x00, 0x00, 0x01}
	StartCode4 = []byte{0x00, 0x00, 0x00, 0x01}
)

func ExtraDataToAnnexB(src []byte) ([]byte, int, error) {
	dstBuffer := libbufio.NewByteBuffer()
	buffer := libbufio.NewByteBuffer(src)
	buffer.Skip(21)
	lengthSize := buffer.ReadUInt8()&3 + 1
	arrays := int(buffer.ReadUInt8())
	for i := 0; i < arrays; i++ {
		t := HEVCNALUnitType(buffer.ReadUInt8() & 0x3F)
		count := int(buffer.ReadUInt16())
		if t != HevcNalVPS && t != HevcNalSPS && t != HevcNalPPS && t != HevcNalSeiPPrefix && t != HevcNalSeiSuffix {
			return nil, -1, fmt.Errorf("invalid NAL unit type in extradata:%d", t)
		}
		for j := 0; j < count; j++ {
			nalUnitLen := int(buffer.ReadUInt16())
			dstBuffer.Write(StartCode4)
			offset := buffer.ReadOffset()
			dstBuffer.Write(src[offset : offset+nalUnitLen])
			buffer.Skip(nalUnitLen)
		}
	}

	return dstBuffer.ToBytes(), int(lengthSize), nil
}

func Mp4ToAnnexB(dst []byte, data, extra []byte, lengthSize int) (int, error) {
	var n int
	length, index := len(data), 0
	gotIRAP := 0
	extraSize := len(extra)

	for index < length {
		if length-index < lengthSize {
			return -1, fmt.Errorf("invalid data")
		}

		var nalUnitSize int
		var nalUnitType int
		var isIRAP bool
		var addExtraData bool

		for i := 0; i < lengthSize; i++ {
			nalUnitSize = (nalUnitSize << 8) | int(data[index])
			index++
		}

		if nalUnitSize < 2 || nalUnitSize > length-index {
			return -1, fmt.Errorf("invalid data")
		}

		nalUnitType = int(data[index]>>1) & 0x3F
		/* prepend extradata to IRAP frames */
		isIRAP = nalUnitType >= 16 && nalUnitType <= 23
		addExtraData = isIRAP && gotIRAP == 0
		if isIRAP {
			gotIRAP |= 1
		} else {
			gotIRAP |= 0
		}

		if addExtraData && extraSize > 0 {
			copy(dst[n:], extra)
			n += extraSize
		}

		copy(dst[n:], StartCode4)
		n += 4

		copy(dst[n:], data[index:index+nalUnitSize])
		n += nalUnitSize
		index += nalUnitSize
	}

	return n, nil
}
