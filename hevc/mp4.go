package hevc

import (
	"encoding/binary"
	"fmt"
	"github.com/lkmio/avformat/avc"
	"github.com/lkmio/avformat/bufio"
)

/*
ISO/IEC 14496-15:2014
8.3.3.1.2   Syntax
aligned(8) class HEVCDecoderConfigurationRecord {
unsigned int(8) configurationVersion = 1;
unsigned int(2) general_profile_space;
unsigned int(1) general_tier_flag;
unsigned int(5) general_profile_idc;
unsigned int(32) general_profile_compatibility_flags;
unsigned int(48) general_constraint_indicator_flags;
unsigned int(8) general_level_idc;
bit(4) reserved = ‘1111’b;
unsigned int(12) min_spatial_segmentation_idc;
bit(6) reserved = ‘111111’b;
unsigned int(2) parallelismType;
bit(6) reserved = ‘111111’b;
unsigned int(2) chroma_format_idc;
bit(5) reserved = ‘11111’b;
unsigned int(3) bit_depth_luma_minus8;
bit(5) reserved = ‘11111’b;
unsigned int(3) bit_depth_chroma_minus8;
bit(16) avgFrameRate;
bit(2) constantFrameRate;
bit(3) numTemporalLayers;
bit(1) temporalIdNested;
unsigned int(2) lengthSizeMinusOne;
unsigned int(8) numOfArrays;
for (j=0; j < numOfArrays; j++) {
bit(1) array_completeness;
unsigned int(1) reserved = 0;
unsigned int(6) NAL_unit_type;
unsigned int(16) numNalus;
for (i=0; i< numNalus; i++) {
unsigned int(16) nalUnitLength;
bit(8*nalUnitLength) nalUnit;
}
}
}
*/

type HEVCDecoderConfigurationRecord struct {
	ConfigurationVersion             byte
	GeneralProfileSpace              byte
	GeneralTierFlag                  byte
	GeneralProfileIdc                byte
	GeneralProfileCompatibilityFlags uint32
	GeneralConstraintIndicatorFlags  uint64
	GeneralLevelIdc                  byte
	MinSpatialSegmentationIdc        uint16
	ParallelismType                  byte
	ChromaFormat                     byte
	BitDepthLumaMinus8               byte
	BitDepthChromaMinus8             byte
	AvgFrameRate                     uint16
	ConstantFrameRate                byte
	NumTemporalLayers                byte
	TemporalIdNested                 byte
	LengthSizeMinusOne               byte
	NumOfArrays                      byte

	VPSList [][]byte
	SPSList [][]byte
	PPSList [][]byte
}

func (r *HEVCDecoderConfigurationRecord) Marshal(vpsList, spsList, ppsList [][]byte) ([]byte, error) {
	if len(spsList) == 0 {
		return nil, fmt.Errorf("sps cannot be null")
	}
	if len(ppsList) == 0 {
		return nil, fmt.Errorf("pps cannot be null")
	}
	if len(vpsList) == 0 {
		return nil, fmt.Errorf("vps cannot be null")
	}

	bytes := make([]byte, 1024)
	bytes[0] = 1
	bytes[21] = 3
	bytes[22] = 3
	writer := bufio.NewBytesWriter(bytes)
	if err := writer.Seek(23); err != nil {
		return nil, err
	}

	write := func(data [][]byte) error {
		noStartCodeData := avc.RemoveStartCode(data[0])
		if err := writer.WriteUint8((noStartCodeData[0] >> 1) & 0x3F); err != nil {
			return err
		}

		if err := writer.WriteUint16(uint16(len(data))); err != nil {
			return err
		}

		for _, i2 := range data {
			noStartCodeData = avc.RemoveStartCode(i2)
			if err := writer.WriteUint16(uint16(len(noStartCodeData))); err != nil {
				return err
			}

			if err := writer.Write(noStartCodeData); err != nil {
				return err
			}
		}
		return nil
	}

	if err := write(vpsList); err != nil {
		return nil, err
	}

	if err := write(spsList); err != nil {
		return nil, err
	}

	if err := write(ppsList); err != nil {
		return nil, err
	}

	return bytes[:writer.Offset()], nil
}

func (r *HEVCDecoderConfigurationRecord) Unmarshal(data []byte) error {
	reader := bufio.NewBytesReader(data)
	if err := reader.Seek(23); err != nil {
		return err
	}

	r.ConfigurationVersion = data[0]
	r.GeneralProfileSpace = data[1] >> 6 & 0x3
	r.GeneralTierFlag = data[1] >> 5 & 0x1
	r.GeneralProfileIdc = data[1] & 0x1F
	r.GeneralProfileCompatibilityFlags = binary.BigEndian.Uint32(data[2:])
	r.GeneralConstraintIndicatorFlags = uint64(binary.BigEndian.Uint32(data[6:])) << 16
	r.GeneralConstraintIndicatorFlags |= uint64(binary.BigEndian.Uint16(data[10:]))
	r.GeneralLevelIdc = data[12]
	r.MinSpatialSegmentationIdc = binary.BigEndian.Uint16(data[13:]) & 0x0FFF
	r.ParallelismType = data[15] & 0x3
	r.ChromaFormat = data[16] & 0x3
	r.BitDepthLumaMinus8 = data[17] & 0x7
	r.BitDepthChromaMinus8 = data[18] & 0x7
	r.AvgFrameRate = binary.BigEndian.Uint16(data[19:])

	r.ConstantFrameRate = data[21] >> 6 & 0x3
	r.NumTemporalLayers = data[21] >> 3 & 0x7
	r.TemporalIdNested = data[21] >> 2 & 0x1
	r.LengthSizeMinusOne = data[21]&0x3 + 1

	r.NumOfArrays = data[22]

	for i := 0; i < int(r.NumOfArrays); i++ {
		readUint, err := reader.ReadUint8()
		if err != nil {
			return err
		}

		naluCount, err := reader.ReadUint16()
		if err != nil {
			return err
		}

		headerType := readUint & 0x3F
		for j := 0; j < int(naluCount); j++ {
			naluLength, err := reader.ReadUint16()
			if err != nil {
				return err
			}

			bytes, err := reader.ReadBytes(int(naluLength))
			if err != nil {
				return err
			}

			//添加start code
			nalu := make([]byte, len(bytes)+4)
			binary.BigEndian.PutUint32(nalu, 0x1)
			copy(nalu[4:], bytes)

			if HevcNalVPS == HEVCNALUnitType(headerType) {
				r.VPSList = append(r.VPSList, nalu)
			} else if HevcNalSPS == HEVCNALUnitType(headerType) {
				r.SPSList = append(r.PPSList, nalu)
			} else if HevcNalPPS == HEVCNALUnitType(headerType) {
				r.PPSList = append(r.PPSList, nalu)
			}
		}
	}

	if len(r.SPSList) == 0 {
		return fmt.Errorf("h265parser: no SPS found in HEVCDecoderConfRecord")
	}
	if len(r.PPSList) == 0 {
		return fmt.Errorf("h265parser: no PPS found in HEVCDecoderConfRecord")
	}
	if len(r.VPSList) == 0 {
		return fmt.Errorf("h265parser: no VPS found in HEVCDecoderConfRecord")
	}
	return nil
}

func ExtraDataToAnnexB(data []byte) ([]byte, error) {
	record := HEVCDecoderConfigurationRecord{}
	if err := record.Unmarshal(data); err != nil {
		return nil, err
	}

	bytes := make([]byte, 1024)
	writer := bufio.NewBytesWriter(bytes)

	for _, sps := range record.VPSList {
		if err := writer.Write(sps); err != nil {
			return nil, err
		}
	}

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
