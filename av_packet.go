package avformat

import (
	"github.com/lkmio/avformat/utils"
	"sync"
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

var (
	PacketPool = sync.Pool{
		New: func() any {
			return &AVPacket{}
		},
	}
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

	PacketType    PacketType // 视频打包模式
	dataAVCC      []byte
	dataAnnexB    []byte
	OnBufferAlloc func(size int) []byte
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
	packet := PacketPool.Get().(*AVPacket)
	packet.Data = data
	packet.Dts = ts
	packet.Pts = ts
	packet.CodecID = id
	packet.Index = index
	packet.Timebase = timebase
	packet.MediaType = utils.AVMediaTypeAudio

	return packet
}

func NewVideoPacket(data []byte, dts, pts int64, key bool, pktType PacketType, id utils.AVCodecID, index, timebase int) *AVPacket {
	packet := PacketPool.Get().(*AVPacket)
	packet.Data = data
	packet.Dts = dts
	packet.Pts = pts
	packet.Key = key
	packet.PacketType = pktType
	packet.CodecID = id
	packet.Index = index
	packet.Timebase = timebase
	packet.MediaType = utils.AVMediaTypeVideo

	return packet
}

func FreePacket(packet *AVPacket) {
	packet.Data = nil
	packet.dataAVCC = nil
	packet.dataAnnexB = nil
	packet.OnBufferAlloc = nil
	packet.BufferIndex = 0
	packet.Dts = 0
	packet.Pts = 0
	packet.Duration = 0
	packet.Key = false
	packet.CreatedTime = 0
	packet.Index = 0
	packet.Timebase = 0
	packet.PacketType = PacketTypeNONE
	packet.MediaType = utils.AVMediaTypeUnknown
	packet.CodecID = utils.AVCodecIdNONE
	PacketPool.Put(packet)
}
