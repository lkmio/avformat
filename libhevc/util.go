package libhevc

import (
	"fmt"
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/utils"
)

func ExtraDataToAnnexB(src []byte) ([]byte, int, error) {
	dstBuffer := utils.NewByteBuffer()
	buffer := utils.NewByteBuffer(src)
	buffer.Skip(21)
	lengthSize := buffer.ReadUInt8()&3 + 1
	arrays := int(buffer.ReadUInt8())
	for i := 0; i < arrays; i++ {
		t := HEVCNALUnitType(buffer.ReadUInt8() & 0x3F)
		count := int(buffer.ReadUInt16())
		if t != HevcNalVPS && t != HevcNalSPS && t != HevcNalPPS && t != HevcNalSeiPPrefix && t != HevcNalSeiSuffix {
			return nil, -1, fmt.Errorf("invalid data")
		}
		for j := 0; j < count; j++ {
			nalUnitLen := int(buffer.ReadUInt16())
			dstBuffer.Write(libavc.StartCode4)
			offset := buffer.ReadOffset()
			dstBuffer.Write(src[offset : offset+nalUnitLen])
			buffer.Skip(nalUnitLen)
		}
	}

	return dstBuffer.ToBytes(), int(lengthSize), nil
}

func Mp4ToAnnexB(dst utils.ByteBuffer, data, extra []byte, lengthSize int) error {
	length, index := len(data), 0
	gotIRAP := 0
	extraSize := len(extra)
	for index < length {
		if length-index < lengthSize {
			return fmt.Errorf("invalid data")
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
			return fmt.Errorf("invalid data")
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
			dst.Write(extra)
		}

		dst.Write(libavc.StartCode4)
		dst.Write(data[index : index+nalUnitSize])
		index += nalUnitSize
	}

	return nil
}
