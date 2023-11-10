package libavc

import (
	"encoding/binary"
	"github.com/yangjiechina/avformat/utils"
)

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

func Mp4ToAnnexB(buffer utils.ByteBuffer, data, extra []byte) {
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

func ExtraDataToAnnexB(src []byte) ([]byte, error) {
	buffer := utils.NewByteBuffer(src)
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
	dstBuffer := utils.NewByteBuffer()
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

		bytes := buffer.ReadableBytes()
		spsDone++
		if bytes > 2 && unitNb == 0 && spsDone == 1 {
			if err := buffer.PeekCount(1); err != nil {
				return nil, err
			}
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
