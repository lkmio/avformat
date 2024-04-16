package utils

import (
	"fmt"
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/libhevc"
	"time"
)

type SPSInfo interface {
	Id() uint
	ProfileIdc() uint
	LevelIdc() uint
	ConstraintSetFlag() uint
	MbWidth() uint
	MbHeight() uint
	CropLeft() uint
	CropRight() uint
	CropTop() uint
	CropBottom() uint
	Width() uint
	Height() uint
	FPS() uint
}

type DecoderConfRecord interface {
	Profile() uint8

	Compatibility() uint8

	Level() uint8

	LengthSize() uint8

	SPSBytes() [][]byte

	PPSBytes() [][]byte
}

type HEVCDecoderConfRecord interface {
	VPSBytes() [][]byte
}

type CodecData interface {
	Record() []byte

	DecoderConfRecord() DecoderConfRecord

	SPSInfo() SPSInfo
}

func ParseAVCDecoderConfigurationRecord(data []byte) (CodecData, error) {
	record, info, err := libavc.NewCodecDataFromAVCDecoderConfRecord(data)
	if err != nil {
		return nil, err
	}

	return &codecData{data, record, info}, nil
}

func ParseHEVCDecoderConfigurationRecord(data []byte) (CodecData, error) {
	record, info, err := libhevc.NewCodecDataFromAVCDecoderConfRecord(data)
	if err != nil {
		return nil, err
	}

	return &codecData{data, record, info}, nil
}

type codecData struct {
	record     []byte
	recordInfo DecoderConfRecord
	spsInfo    SPSInfo
}

func (self codecData) Record() []byte {
	return self.record
}

func (self codecData) DecoderConfRecord() DecoderConfRecord {
	return self.recordInfo
}

func (self codecData) SPSInfo() SPSInfo {
	return self.spsInfo
}

func (self codecData) SPS() []byte {
	if len(self.recordInfo.SPSBytes()) > 0 {
		return self.recordInfo.SPSBytes()[0]
	}

	return []byte{0}
}

func (self codecData) PPS() []byte {
	if len(self.recordInfo.PPSBytes()) > 0 {
		return self.recordInfo.PPSBytes()[0]
	}

	return []byte{0}
}

func (self codecData) Width() int {
	return int(self.spsInfo.Width())
}

func (self codecData) Height() int {
	return int(self.spsInfo.Height())
}

func (self codecData) FPS() int {
	return int(self.spsInfo.FPS())
}

func (self codecData) Resolution() string {
	return fmt.Sprintf("%vx%v", self.Width(), self.Height())
}

func (self codecData) Tag() string {
	return fmt.Sprintf("avc1.%02X%02X%02X", self.recordInfo.Profile(), self.recordInfo.Compatibility(), self.recordInfo.Level())
}

func (self codecData) Bandwidth() string {
	return fmt.Sprintf("%v", (int(float64(self.Width())*(float64(1.71)*(30/float64(self.FPS())))))*1000)
}

func (self codecData) PacketDuration(data []byte) time.Duration {
	return time.Duration(1000./float64(self.FPS())) * time.Millisecond
}
