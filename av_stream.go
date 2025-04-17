package avformat

import "github.com/lkmio/avformat/utils"

type AudioConfig struct {
	SampleRate    int  // 音频采样率
	SampleSize    int  // 音频采样位深
	Channels      int  // 音频通道数
	HasADTSHeader bool // 是否存在ADTSHeader
}

type AVStream struct {
	MediaType       utils.AVMediaType
	Index           int
	CodecID         utils.AVCodecID
	CodecParameters CodecData
	Colors          []byte
	Data            []byte
	Timebase        int

	AudioConfig
}

func NewAVStream(type_ utils.AVMediaType, index int, codecId utils.AVCodecID, extra []byte, config CodecData) *AVStream {
	return &AVStream{MediaType: type_, Index: index, CodecID: codecId, Data: extra, CodecParameters: config}
}
