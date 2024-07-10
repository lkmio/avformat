package utils

import (
	"github.com/lkmio/avformat/libavc"
	"github.com/lkmio/avformat/libhevc"
)

// CreateHevcStreamFromKeyFrame 从关键帧中提取sps和pps创建AVStream
func CreateHevcStreamFromKeyFrame(data []byte, index int) (AVStream, error) {
	vps, sps, pps, err := libhevc.ParseExtraDataFromKeyNALU(data)
	if err != nil {
		return nil, err
	}

	codecData, err := NewHEVCCodecData(vps, sps, pps)
	if err != nil {
		return nil, err
	}

	return NewAVStream(AVMediaTypeVideo, index, AVCodecIdH265, codecData.AnnexBExtraData(), codecData), nil
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

	return NewAVStream(AVMediaTypeVideo, index, AVCodecIdH264, codecData.AnnexBExtraData(), codecData), nil
}
