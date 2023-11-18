package utils

type PacketType byte

const (
	PacketTypeAnnexB = PacketType(1)
	PacketTypeAVCC   = PacketType(2)
	PacketTypeNONE   = PacketType(3)
)

type AVPacket interface {
	Data() []byte

	Pts() int64

	Dts() int64

	KeyFrame() bool

	PacketType() PacketType //打包模式

	MediaType() AVMediaType //冗余媒体类型

	CodecId() AVCodecID //冗余编码器Id
}

type avPacket struct {
	data []byte
	dts  int64
	pts  int64
	key  bool

	//打包模式
	packetType PacketType
	//冗余媒体类型
	mediaType AVMediaType
	//冗余编码器Id
	codecId AVCodecID
}

func NewAudioPacket(data []byte, dts, pts int64) AVPacket {
	return &avPacket{data: data, dts: dts, pts: pts, key: true}
}

func NewVideoPacket(data []byte, dts, pts int64, key bool, packetType PacketType, mediaType AVMediaType, id AVCodecID) AVPacket {
	return &avPacket{data: data, dts: dts, pts: pts, key: key, packetType: packetType, mediaType: mediaType, codecId: id}
}

func (pkt *avPacket) Dts() int64 {
	return pkt.dts
}

func (pkt *avPacket) Pts() int64 {
	return pkt.pts
}

func (pkt *avPacket) MediaType() AVMediaType {
	return pkt.mediaType
}

func (pkt *avPacket) Data() []byte {
	return pkt.data
}

func (pkt *avPacket) KeyFrame() bool {
	return pkt.key
}

func (pkt *avPacket) PacketType() PacketType {
	return pkt.packetType
}

func (pkt *avPacket) CodecId() AVCodecID {
	return pkt.codecId
}
