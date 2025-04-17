package avc

import (
	"encoding/binary"
	"fmt"
	"github.com/lkmio/avformat/bufio"
)

/*
aligned(8) class AVCDecoderConfigurationRecord {
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
}
*/

type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion byte
	AVCProfileIndication byte
	ProfileCompatibility byte
	AVCLevelIndication   byte
	LengthSizeMinusOne   byte

	SPSList [][]byte // AnnexB格式
	PPSList [][]byte
}

func (a *AVCDecoderConfigurationRecord) Marshal(spsList, ppsList [][]byte) ([]byte, error) {
	if len(spsList) == 0 {
		return nil, fmt.Errorf("sps cannot be null")
	}
	if len(ppsList) == 0 {
		return nil, fmt.Errorf("pps cannot be null")
	}

	bytes := make([]byte, 1024)
	writer := bufio.NewBytesWriter(bytes)
	if err := writer.Seek(5); err != nil {
		return nil, err
	}

	if err := writer.WriteUint8(byte(len(spsList) & 0x1F)); err != nil {
		return nil, err
	}

	var noStartCodeSps []byte
	for _, sps := range spsList {
		noStartCodeSps = RemoveStartCode(sps)
		if err := writer.WriteUint16(uint16(len(noStartCodeSps))); err != nil {
			return nil, err
		}

		if err := writer.Write(noStartCodeSps); err != nil {
			return nil, err
		}
	}

	if err := writer.WriteUint8(byte(len(ppsList))); err != nil {
		return nil, err
	}

	for _, pps := range ppsList {
		noStartCodePps := RemoveStartCode(pps)
		if err := writer.WriteUint16(uint16(len(noStartCodePps))); err != nil {
			return nil, err
		}

		if err := writer.Write(noStartCodePps); err != nil {
			return nil, err
		}
	}

	bytes[0] = 1
	bytes[1] = noStartCodeSps[3]
	bytes[2] = noStartCodeSps[4]
	bytes[3] = noStartCodeSps[5]
	bytes[4] = 0xff
	bytes[5] = 0xE0 | byte(len(spsList))
	return bytes[:writer.Offset()], nil
}

func (a *AVCDecoderConfigurationRecord) Unmarshal(data []byte) error {
	var err error
	var spsCount uint8
	var ppsCount uint8

	reader := bufio.NewBytesReader(data)

	if err = reader.Seek(5); err != nil {
		return err
	}

	//#define MAX_SPS_COUNT          32
	//#define MAX_PPS_COUNT         256
	spsCount, err = reader.ReadUint8()
	if err != nil {
		return err
	}
	spsCount &= 0x1F
	if spsCount < 1 {
		return fmt.Errorf("no sps found")
	}

	for i := 0; i < int(spsCount); i++ {
		length, err := reader.ReadUint16()
		if err != nil {
			return err
		}

		bytes, err := reader.ReadBytes(int(length))
		if err != nil {
			return err
		}

		// 添加start code
		sps := make([]byte, len(bytes)+4)
		binary.BigEndian.PutUint32(sps, 0x1)
		copy(sps[4:], bytes)

		a.SPSList = append(a.SPSList, sps)
	}

	ppsCount, err = reader.ReadUint8()
	if err != nil {
		return err
	}
	if ppsCount < 1 {
		return fmt.Errorf("no pps found")
	}

	for i := 0; i < int(ppsCount); i++ {
		length, err := reader.ReadUint16()
		if err != nil {
			return err
		}

		bytes, err := reader.ReadBytes(int(length))
		if err != nil {
			return err
		}

		// 添加start code
		pps := make([]byte, len(bytes)+4)
		binary.BigEndian.PutUint32(pps, 0x1)
		copy(pps[4:], bytes)

		a.PPSList = append(a.PPSList, pps)
	}

	a.ConfigurationVersion = data[0]
	a.AVCProfileIndication = data[1]
	a.ProfileCompatibility = data[2]
	a.AVCLevelIndication = data[3]
	a.LengthSizeMinusOne = data[4] & 0x3
	return nil
}

// ExtraDataToAnnexB AVCDecoderConfigurationRecord转AnnexB
func ExtraDataToAnnexB(data []byte) ([]byte, error) {
	record := AVCDecoderConfigurationRecord{}
	if err := record.Unmarshal(data); err != nil {
		return nil, err
	}

	bytes := make([]byte, 1024)
	writer := bufio.NewBytesWriter(bytes)

	for _, sps := range record.SPSList {
		if err := writer.Write(sps); err != nil {
			return nil, err
		}
	}

	for _, pps := range record.PPSList {
		if err := writer.Write(pps); err != nil {
			return nil, err
		}
	}

	return bytes[:writer.Offset()], nil
}
