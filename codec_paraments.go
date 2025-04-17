package avformat

import (
	"fmt"
	"github.com/lkmio/avformat/avc"
	"github.com/lkmio/avformat/hevc"
)

type CodecData interface {
	AnnexBExtraData() []byte

	MP4ExtraData() []byte

	Width() int

	Height() int

	SPS() [][]byte

	PPS() [][]byte
}

type codecData struct {
	width  int
	height int
	annexB []byte
	m4vc   []byte
}

func (c *codecData) Width() int {
	return c.width
}

func (c *codecData) Height() int {
	return c.height
}

type AVCCodecData struct {
	codecData
	Record *avc.AVCDecoderConfigurationRecord
}

func (h AVCCodecData) AnnexBExtraData() []byte {
	if h.annexB == nil {
		h.annexB = mix(h.Record.SPSList, h.Record.PPSList)
	}

	return h.annexB
}

func (h AVCCodecData) MP4ExtraData() []byte {
	if h.m4vc == nil {
		h.m4vc, _ = h.Record.Marshal(h.Record.SPSList, h.Record.PPSList)
	}

	return h.m4vc
}

func (h AVCCodecData) SPS() [][]byte {
	return h.Record.SPSList
}

func (h AVCCodecData) PPS() [][]byte {
	return h.Record.PPSList
}

type HEVCCodecData struct {
	codecData
	Record *hevc.HEVCDecoderConfigurationRecord
}

func (h HEVCCodecData) SPS() [][]byte {
	return h.Record.SPSList
}

func (h HEVCCodecData) PPS() [][]byte {
	return h.Record.PPSList
}

func (h HEVCCodecData) VPS() [][]byte {
	return h.Record.VPSList
}

func (h HEVCCodecData) AnnexBExtraData() []byte {
	if h.annexB == nil {
		h.annexB = mix(h.Record.VPSList, h.Record.SPSList, h.Record.PPSList)
	}

	return h.annexB
}

func (h HEVCCodecData) MP4ExtraData() []byte {
	if h.m4vc == nil {
		h.m4vc, _ = h.Record.Marshal(h.Record.VPSList, h.Record.SPSList, h.Record.PPSList)
	}

	return h.m4vc
}

func ParseAVCDecoderConfigurationRecord(data []byte) (CodecData, error) {
	configurationRecord := avc.AVCDecoderConfigurationRecord{}
	if err := configurationRecord.Unmarshal(data); err != nil {
		return nil, err
	}

	sps, err := avc.ParseSPS(configurationRecord.SPSList[0])
	if err != nil {
		return nil, err
	}

	avcCodecData := AVCCodecData{
		codecData: codecData{
			m4vc:   data,
			width:  sps.Width,
			height: sps.Height,
		},
		Record: &configurationRecord,
	}

	return &avcCodecData, nil
}

func ParseHEVCDecoderConfigurationRecord(data []byte) (CodecData, error) {
	configurationRecord := hevc.HEVCDecoderConfigurationRecord{}
	if err := configurationRecord.Unmarshal(data); err != nil {
		return nil, err
	}

	sps, err := hevc.ParseSPS(configurationRecord.SPSList[0])
	if err != nil {
		return nil, err
	}

	c := HEVCCodecData{
		codecData: codecData{
			m4vc:   data,
			width:  sps.Width,
			height: sps.Height,
		},
		Record: &configurationRecord,
	}
	return &c, nil
}

func mix(data ...[][]byte) []byte {
	var extra []byte
	for _, v := range data {
		for _, bytes := range v {
			extra = append(extra, bytes...)
		}
	}

	return extra
}

func NewAVCCodecData(sps, pps []byte) (CodecData, error) {
	spsInfo, err := avc.ParseSPS(sps)
	if err != nil {
		return nil, fmt.Errorf("h264parser: parse SPS failed(%s)", err)
	}

	recordInfo := avc.AVCDecoderConfigurationRecord{
		SPSList: make([][]byte, 1),
		PPSList: make([][]byte, 1),
	}

	recordInfo.SPSList[0] = sps
	recordInfo.PPSList[0] = pps

	c := AVCCodecData{codecData: codecData{
		annexB: mix(recordInfo.SPSList, recordInfo.PPSList),
		width:  spsInfo.Width,
		height: spsInfo.Height,
	},
		Record: &recordInfo,
	}

	return &c, nil
}

func NewHEVCCodecData(vps, sps, pps []byte) (CodecData, error) {
	spsInfo, err := hevc.ParseSPS(sps)
	if err != nil {
		return nil, fmt.Errorf("h265parser: parse SPS failed(%s)", err)
	}

	recordInfo := hevc.HEVCDecoderConfigurationRecord{
		VPSList: make([][]byte, 1),
		SPSList: make([][]byte, 1),
		PPSList: make([][]byte, 1),
	}

	recordInfo.VPSList[0] = vps
	recordInfo.SPSList[0] = sps
	recordInfo.PPSList[0] = pps

	c := HEVCCodecData{codecData: codecData{
		annexB: mix(recordInfo.VPSList, recordInfo.SPSList, recordInfo.PPSList),
		width:  spsInfo.Width,
		height: spsInfo.Height,
	},
		Record: &recordInfo,
	}

	return &c, nil
}
