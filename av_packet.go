package avformat

import (
	"github.com/lkmio/avformat/utils"
)

type PacketType byte

type DTS int64

type PTS int64

func (d DTS) Value() int64 {
	return int64(d)
}

func (p PTS) Value() int64 {
	return int64(p)
}

const (
	PacketTypeAnnexB = PacketType(1)
	PacketTypeAVCC   = PacketType(2)
	PacketTypeNONE   = PacketType(3)
	DTSUndefined     = DTS(-1)
)

type AVPacket struct {
	Data        []byte
	Pts         int64
	Dts         int64
	Duration    int64
	Key         bool
	CreatedTime int64 // 创建Packet的Unix时间
	BufferIndex int   // Data在内存池中的索引

	Index     int
	Timebase  int
	MediaType utils.AVMediaType // 冗余媒体类型
	CodecID   utils.AVCodecID   // 冗余编码器ID

	PacketType PacketType // 视频打包模式
	dataAVCC   []byte
	dataAnnexB []byte
}

func (pkt *AVPacket) ConvertDts(dstTimebase int) int64 {
	return ConvertTs(pkt.Dts, pkt.Timebase, dstTimebase)
}

func (pkt *AVPacket) ConvertPts(dstTimebase int) int64 {
	return ConvertTs(pkt.Pts, pkt.Timebase, dstTimebase)
}

func (pkt *AVPacket) GetDuration(timebase int) int64 {
	if pkt.Timebase == timebase {
		return pkt.Duration
	}

	return ConvertTs(pkt.Duration, pkt.Timebase, timebase)
}

func NewAudioPacket(data []byte, ts int64, id utils.AVCodecID, index, timebase int) *AVPacket {
	packet := &AVPacket{
		Data:      data,
		Dts:       ts,
		Pts:       ts,
		CodecID:   id,
		Index:     index,
		Timebase:  timebase,
		MediaType: utils.AVMediaTypeAudio,
	}

	return packet
}

func NewVideoPacket(data []byte, dts, pts int64, key bool, pktType PacketType, id utils.AVCodecID, index, timebase int) *AVPacket {
	packet := &AVPacket{
		Data:       data,
		Dts:        dts,
		Pts:        pts,
		Key:        key,
		PacketType: pktType,
		CodecID:    id,
		Index:      index,
		Timebase:   timebase,
		MediaType:  utils.AVMediaTypeVideo,
	}

	return packet
}
