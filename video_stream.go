package avformat

import (
	"encoding/hex"
	"fmt"
	"github.com/lkmio/avformat/avc"
	"github.com/lkmio/avformat/hevc"
	"github.com/lkmio/avformat/utils"
)

// CreateHevcStreamFromKeyFrame 从关键帧中提取sps和pps创建AVStream
func CreateHevcStreamFromKeyFrame(data []byte, index int) (*AVStream, error) {
	vps, sps, pps, err := hevc.ParseExtraDataFromKeyNALU(data)
	if err != nil {
		return nil, err
	}

	codecData, err := NewHEVCCodecData(vps, sps, pps)
	if err != nil {
		return nil, err
	}

	return NewAVStream(utils.AVMediaTypeVideo, index, utils.AVCodecIdH265, codecData.AnnexBExtraData(), codecData), nil
}

// CreateAVCStreamFromKeyFrame 从关键帧中提取sps和pps创建AVStream
func CreateAVCStreamFromKeyFrame(data []byte, index int) (*AVStream, error) {
	sps, pps, err := avc.ParseExtraDataFromKeyNALU(data)
	if err != nil {
		return nil, err
	}

	codecData, err := NewAVCCodecData(sps, pps)
	if err != nil {
		return nil, err
	}

	return NewAVStream(utils.AVMediaTypeVideo, index, utils.AVCodecIdH264, codecData.AnnexBExtraData(), codecData), nil
}

func ExtractVideoExtraDataFromKeyFrame(codec utils.AVCodecID, data []byte) ([]byte, error) {
	if utils.AVCodecIdH264 == codec {
		sps, pps, err := avc.ParseExtraDataFromKeyNALU(data)
		if err != nil {
			fmt.Printf("从关键帧中解析sps pps失败 data:%s \r\n", hex.EncodeToString(data))
			return nil, err
		}

		return append(sps, pps...), nil
	} else if utils.AVCodecIdH265 == codec {
		vps, sps, pps, err := hevc.ParseExtraDataFromKeyNALU(data)
		if err != nil {
			fmt.Printf("从关键帧中解析vps sps pps失败  data:%s \r\n", hex.EncodeToString(data))
			return nil, err
		}

		return append(append(vps, sps...), pps...), nil
	}

	return nil, nil
}

func ExtractAudioExtraData(codec utils.AVCodecID, data []byte) ([]byte, int, AudioConfig, error) {
	if utils.AVCodecIdAAC == codec {
		// 必须包含ADTSHeader
		if len(data) < 7 {
			return nil, -1, AudioConfig{}, fmt.Errorf("need more data")
		}

		var skip int
		header, err := utils.ReadADtsFixedHeader(data)
		if err != nil {
			fmt.Printf("读取ADTSHeader失败 data:%s\r\n", hex.EncodeToString(data[:7]))
			return nil, -1, AudioConfig{}, err
		} else {
			skip = 7
			// 跳过ADtsHeader长度
			if header.ProtectionAbsent() == 0 {
				skip += 2
			}
		}

		extraData, err := utils.ADtsHeader2MpegAudioConfigData(header)
		if err != nil {
			fmt.Printf("adt头转m4ac失败 data:%s\r\n", hex.EncodeToString(data[:7]))
			return nil, -1, AudioConfig{}, err
		}

		rate, _ := utils.GetSampleRateFromFrequency(header.Frequency())

		return extraData, skip, AudioConfig{
			SampleRate:    rate,
			SampleSize:    16,
			Channels:      header.Channel(),
			HasADTSHeader: true,
		}, nil
	} else if utils.AVCodecIdPCMALAW == codec || utils.AVCodecIdPCMMULAW == codec {

	}

	return nil, 0, AudioConfig{
		SampleRate: 8000,
		SampleSize: 16,
		Channels:   1,
	}, nil
}

//func ExtractVideoPacket(codec utils.AVCodecID, key, data []byte, pts, dts int64, index, timebase int) (*AVPacket, error) {
//	var stream *AVStream
//
//	if utils.AVCodecIdH264 == codec {
//		//从关键帧中解析出sps和pps
//		if key && extractStream {
//			sps, pps, err := avc.ParseExtraDataFromKeyNALU(data)
//			if err != nil {
//				fmt.Errorf("从关键帧中解析sps pps失败 data:%s \r\n", hex.EncodeToString(data))
//				return nil, nil, err
//			}
//
//			codecData, err := NewAVCCodecData(sps, pps)
//			if err != nil {
//				fmt.Errorf("解析sps pps失败 data:%s sps:%s, pps:%s \r\n", hex.EncodeToString(data), hex.EncodeToString(sps), hex.EncodeToString(pps))
//				return nil, nil, err
//			}
//
//			stream = NewAVStream(utils.AVMediaTypeVideo, 0, codec, codecData.AnnexBExtraData(), codecData)
//		}
//
//	} else if utils.AVCodecIdH265 == codec {
//		if key && extractStream {
//			vps, sps, pps, err := hevc.ParseExtraDataFromKeyNALU(data)
//			if err != nil {
//				fmt.Errorf("从关键帧中解析vps sps pps失败  data:%s \r\n", hex.EncodeToString(data))
//				return nil, nil, err
//			}
//
//			codecData, err := NewHEVCCodecData(vps, sps, pps)
//			if err != nil {
//				fmt.Errorf("解析sps pps失败 data:%s vps:%s sps:%s, pps:%s \r\n", hex.EncodeToString(data), hex.EncodeToString(vps), hex.EncodeToString(sps), hex.EncodeToString(pps))
//				return nil, nil, err
//			}
//
//			stream = NewAVStream(utils.AVMediaTypeVideo, 0, codec, codecData.AnnexBExtraData(), codecData)
//		}
//
//	}
//
//	packet := NewVideoPacket(data, dts, pts, key, PacketTypeAnnexB, codec, index, timebase)
//	return stream, packet, nil
//}

func ExtractAudioPacket(codec utils.AVCodecID, data []byte, ts int64, index, timebase int, adts bool) (*AVPacket, error) {
	var packet *AVPacket
	if utils.AVCodecIdAAC == codec && adts {
		// 必须包含ADTSHeader
		//if len(data) < 7 {
		//	return nil, fmt.Errorf("need more data")
		//}

		var skip int
		//header, err := utils.ReadADtsFixedHeader(data)
		//if err != nil {
		//	fmt.Printf("读取ADTSHeader失败 data:%s\r\n", hex.EncodeToString(data[:7]))
		//	return nil, err
		//} else {
		//	// 跳过ADtsHeader
		//	skip = 7
		//	if header.ProtectionAbsent() == 0 {
		//		skip += 2
		//	}
		//}

		packet = NewAudioPacket(data[skip:], ts, codec, index, timebase)
	} else /*if utils.AVCodecIdPCMALAW == codec || utils.AVCodecIdPCMMULAW == codec*/ {

		packet = NewAudioPacket(data, ts, codec, index, timebase)
	}

	return packet, nil
}

func generateVideoCodecData(packType PacketType, id utils.AVCodecID, data []byte) (CodecData, error) {
	if packType == PacketTypeAVCC {
		switch id {
		case utils.AVCodecIdH264:
			return ParseAVCDecoderConfigurationRecord(data)
		case utils.AVCodecIdH265:
			return ParseHEVCDecoderConfigurationRecord(data)
		}
	} else {
		switch id {
		case utils.AVCodecIdH264:
			sps, pps, err := avc.ParseExtraDataFromKeyNALU(data)
			if err != nil {
				return nil, err
			}
			return NewAVCCodecData(sps, pps)
		case utils.AVCodecIdH265:
			vps, sps, pps, err := hevc.ParseExtraDataFromKeyNALU(data)
			if err != nil {
				return nil, err
			}
			return NewHEVCCodecData(vps, sps, pps)
		}
	}

	//return nil, fmt.Errorf("unsupported codec: %s", id)
	return nil, nil
}
