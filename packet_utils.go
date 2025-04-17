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
		return pkt.Data
	} else if pkt.dataAnnexB == nil {
		var n int
		bytes := make([]byte, len(pkt.Data)+256)
		if utils.AVCodecIdH264 == pkt.CodecID {
			n = avc.AVCC2AnnexB(bytes, pkt.Data, nil)
		} else if utils.AVCodecIdH265 == pkt.CodecID {
			var err error

			lengthSize := stream.CodecParameters.(*HEVCCodecData).Record.LengthSizeMinusOne
			n, err = hevc.Mp4ToAnnexB(bytes, pkt.Data, nil, int(lengthSize))
			if err != nil {
				panic(err)
			}
		}

		pkt.dataAnnexB = bytes[:n]
	}

	return pkt.dataAnnexB
}

func AnnexBPacket2AVCC(pkt *AVPacket) []byte {
	utils.Assert(utils.AVMediaTypeVideo == pkt.MediaType)

	if PacketTypeAnnexB != pkt.PacketType || (pkt.CodecID != utils.AVCodecIdH264 && pkt.CodecID != utils.AVCodecIdH265) {
		return pkt.Data
	} else if pkt.dataAVCC == nil {
		bytes := make([]byte, len(pkt.Data)+256)
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
