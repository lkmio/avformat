package utils

import (
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/libhevc"
)

// CreateHevcStreamFromKeyFrame 从关键帧中提取sps和pps创建AVStream
func CreateHevcStreamFromKeyFrame(data []byte, index int) (AVStream, error) {
	vps, sps, pps, err := libhevc.ParseExtraDataFromKeyNALU(data)
	if err != nil {
		return nil, err
	}

	codecData, err := NewHevcCodecData(vps, sps, pps)
	if err != nil {
		return nil, err
	}

	return NewAVStream(AVMediaTypeVideo, index, AVCodecIdH265, codecData.Record(), codecData), nil
}

// CreateAVCStreamFromKeyFrame 从关键帧中提取sps和pps创建AVStream
func CreateAVCStreamFromKeyFrame(data []byte, index int) (AVStream, error) {
	sps, pps, err := libavc.ParseExtraDataFromKeyNALU(data)
	if err != nil {
		return nil, err
	}

	codecData, err := NewAVCCodecData(sps, pps)
	if err != nil {
		return nil, err
	}

	return NewAVStream(AVMediaTypeVideo, index, AVCodecIdH264, codecData.Record(), codecData), nil
}
