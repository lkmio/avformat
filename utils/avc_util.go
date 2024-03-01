package utils

import "encoding/binary"

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

func FindStartCodeFromBuffer(buffer ByteBuffer, offset int) int {
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

func IsKeyFrameFromBuffer(buffer ByteBuffer) bool {
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
			WriteUInt24(dst[offset:], 0x1)
			offset += 3
		}
	}

	copy(dst[offset:], data)
	return startCodeSize + len(data)
}

func copyNalU(buffer ByteBuffer, data []byte, outSize int, append bool) int {
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

type MPEG4AVCConfig struct {
	Version       byte
	Profile       byte
	Compatibility byte
	Level         byte
	LengthSize    byte
	SpsNum        byte
	PpsNum        byte
	Sps           [][]byte
	Pps           [][]byte

	ChromaFormat                 byte //2 bits
	BitDepthLumaMinus8           byte //3 bits
	BitDepthChromaMinus8         byte //3 bits
	NumOfSequenceParameterSetExt byte //8 bits
	SpsExtNALUnit                [][]byte
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
			//case H264NalSEI:
			//	index += size
			//	continue
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
	nalStart := FindStartCode(annexB, 0)
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

		nalEnd := FindStartCode(annexB, nalStart)
		if nalEnd < 0 {
			return size
		}

		nalSize := nalEnd - nalStart
		binary.BigEndian.PutUint32(dst[size:], uint32(nalSize))
		size += 4
		copy(dst[size:], annexB[nalStart:nalEnd])
		size += nalSize
		nalStart = nalEnd
	}
}

func Mp4ToAnnexB(buffer ByteBuffer, data, extra []byte) {
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
	buffer := NewByteBuffer(src)
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
	dstBuffer := NewByteBuffer()
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

/*aligned(8) class AVCDecoderConfigurationRecord {
unsigned int(8) configurationVersion = 1;
unsigned int(8) AVCProfileIndication;
unsigned int(8) profile_compatibility;
unsigned int(8) AVCLevelIndication;
bit(6) reserved = ‘111111’b;
unsigned int(2) lengthSizeMinusOne;
bit(3) reserved = ‘111’b;
unsigned int(5) numOfSequenceParameterSets;
for (i=0; i< numOfSequenceParameterSets; i++) {
unsigned int(16) sequenceParameterSetLength ;
bit(8*sequenceParameterSetLength) sequenceParameterSetNALUnit;
}
unsigned int(8) numOfPictureParameterSets;
for (i=0; i< numOfPictureParameterSets; i++) {
unsigned int(16) pictureParameterSetLength;
bit(8*pictureParameterSetLength) pictureParameterSetNALUnit;
}
if( profile_idc == 100 || profile_idc == 110 ||
profile_idc == 122 || profile_idc == 144 )
{
bit(6) reserved = ‘111111’b;
unsigned int(2) chroma_format;
bit(5) reserved = ‘11111’b;
unsigned int(3) bit_depth_luma_minus8;
bit(5) reserved = ‘11111’b;
unsigned int(3) bit_depth_chroma_minus8;
unsigned int(8) numOfSequenceParameterSetExt;
for (i=0; i< numOfSequenceParameterSetExt; i++) {
unsigned int(16) sequenceParameterSetExtLength;
bit(8*sequenceParameterSetExtLength) sequenceParameterSetExtNALUnit;
}
}
}*/

func ParseDecoderConfigurationRecord(data []byte) (*MPEG4AVCConfig, error) {
	config := &MPEG4AVCConfig{}
	config.Version = data[0]
	config.Profile = data[1]
	config.Compatibility = data[2]
	config.Level = data[3]
	config.LengthSize = data[4] & 0x3

	spsNum := data[5] & 0x1F
	config.Sps = make([][]byte, spsNum)
	index := 6
	for i := 0; i < int(spsNum); i++ {
		length := int(data[index])<<8 | int(data[index+1])
		index += 2 + length
		bytes := make([]byte, length)
		copy(bytes, data[index-length:index])
		config.Sps[i] = bytes
	}

	ppsNum := data[index]
	config.Pps = make([][]byte, ppsNum)
	index += 1
	for i := 0; i < int(ppsNum); i++ {
		length := int(data[index])<<8 | int(data[index+1])
		index += 2 + length
		bytes := make([]byte, length)
		copy(bytes, data[index-length:index])
		config.Pps[i] = bytes
	}

	return config, nil
}
