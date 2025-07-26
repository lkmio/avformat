package avformat

import (
	"github.com/lkmio/avformat/avc"
	"github.com/lkmio/avformat/hevc"
	"github.com/lkmio/avformat/utils"
)

func ConvertTs(ts int64, srcTimeBase, dstTimeBase int) int64 {
	interval := float64(dstTimeBase) / float64(srcTimeBase)
	return int64(float64(ts) * interval)
}

func AVCCPacket2AnnexB(stream *AVStream, pkt *AVPacket) []byte {
	utils.Assert(utils.AVMediaTypeVideo == pkt.MediaType)

	if PacketTypeAVCC != pkt.PacketType || (stream.CodecID != utils.AVCodecIdH264 && stream.CodecID != utils.AVCodecIdH265) {
		// 确保annexb的关键帧开头是vps/sps/pps
		if pkt.Key && (stream.CodecID == utils.AVCodecIdH264 || stream.CodecID == utils.AVCodecIdH265) && stream.CodecParameters != nil && pkt.dataAnnexB == nil {
			extraData := stream.CodecParameters.AnnexBExtraData()
			if avc.RemoveStartCode(extraData)[0] != avc.RemoveStartCode(pkt.Data)[0] {
				size := len(extraData) + len(pkt.Data)
				var data []byte
				if pkt.OnBufferAlloc != nil {
					data = pkt.OnBufferAlloc(size)
				} else {
					data = make([]byte, size)
				}

				copy(data, extraData)
				copy(data[len(extraData):], pkt.Data)
				pkt.dataAnnexB = data
			}
		}

		if pkt.dataAnnexB != nil {
			return pkt.dataAnnexB
		}

		return pkt.Data
	} else if pkt.dataAnnexB == nil {
		var n int
		var bytes []byte

		var extraDataSize int
		if pkt.Key && stream.CodecParameters != nil {
			extraDataSize = len(stream.CodecParameters.AnnexBExtraData())
		}

		dstSize := extraDataSize + len(pkt.Data) + 256
		if pkt.OnBufferAlloc != nil {
			bytes = pkt.OnBufferAlloc(dstSize)
		} else {
			bytes = make([]byte, dstSize)
		}

		if utils.AVCodecIdH264 == pkt.CodecID {
			n = avc.AVCC2AnnexB(bytes[extraDataSize:], pkt.Data, nil)
		} else if utils.AVCodecIdH265 == pkt.CodecID {
			var err error

			lengthSize := stream.CodecParameters.(*HEVCCodecData).Record.LengthSizeMinusOne
			n, err = hevc.Mp4ToAnnexB(bytes[extraDataSize:], pkt.Data, nil, int(lengthSize))
			if err != nil {
				panic(err)
			}
		}

		copy(bytes[:extraDataSize], stream.CodecParameters.AnnexBExtraData())
		n += extraDataSize
		pkt.dataAnnexB = bytes[:n]
	}

	return pkt.dataAnnexB
}

func AnnexBPacket2AVCC(pkt *AVPacket) []byte {
	utils.Assert(utils.AVMediaTypeVideo == pkt.MediaType)

	if PacketTypeAnnexB != pkt.PacketType || (pkt.CodecID != utils.AVCodecIdH264 && pkt.CodecID != utils.AVCodecIdH265) {
		return pkt.Data
	} else if pkt.dataAVCC == nil {
		var bytes []byte
		dstSize := len(pkt.Data) + 256
		if pkt.OnBufferAlloc != nil {
			bytes = pkt.OnBufferAlloc(dstSize)
		} else {
			bytes = make([]byte, dstSize)
		}

		n := avc.AnnexB2AVCC(bytes, pkt.Data)
		pkt.dataAVCC = bytes[:n]
	}

	return pkt.dataAVCC
}

func IsKeyFrame(id utils.AVCodecID, data []byte) bool {
	if utils.AVCodecIdH264 == id {
		return avc.IsKeyFrame(data)
	} else if utils.AVCodecIdH265 == id {
		return hevc.IsKeyFrame(data)
	} else {
		return false
	}
}
