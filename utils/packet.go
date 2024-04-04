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

	AnnexBPacketData() []byte

	AVCCPacketData() []byte

	Index() int

	ConvertPts(dst int) int64

	ConvertDts(dst int) int64

	SetDuration(duration int64)

	Duration(timebase int) int64
}

type avPacket struct {
	data           []byte
	dataAVCC       []byte
	dataAVCCSize   int
	dataAnnexB     []byte
	dataAnnexBSize int

	dts      int64
	pts      int64
	duration int64
	key      bool

	//打包模式
	packetType PacketType
	//冗余媒体类型
	mediaType AVMediaType
	//冗余编码器Id
	codecId AVCodecID

	index int

	timebase int
}

func NewAudioPacket(data []byte, dts, pts int64, id AVCodecID, index int, timebase int) AVPacket {
	return &avPacket{data: data, dts: dts, pts: pts, key: true, mediaType: AVMediaTypeAudio, codecId: id, index: index, timebase: timebase}
}

func NewVideoPacket(data []byte, dts, pts int64, key bool, packetType PacketType, id AVCodecID, index int, timebase int) AVPacket {
	return &avPacket{data: data, dts: dts, pts: pts, key: key, packetType: packetType, mediaType: AVMediaTypeVideo, codecId: id, index: index, timebase: timebase}
}

func ConvertTs(ts int64, srcTimeBase, dstTimeBase int) int64 {
	interval := float64(dstTimeBase) / float64(srcTimeBase)
	return int64(float64(ts) * interval)
}

func (pkt *avPacket) Dts() int64 {
	return pkt.dts
}

func (pkt *avPacket) ConvertDts(dst int) int64 {
	return ConvertTs(pkt.dts, pkt.timebase, dst)
}

func (pkt *avPacket) Pts() int64 {
	return pkt.pts
}

func (pkt *avPacket) ConvertPts(dst int) int64 {
	return ConvertTs(pkt.pts, pkt.timebase, dst)
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

func (pkt *avPacket) AnnexBPacketData() []byte {
	Assert(AVMediaTypeVideo == pkt.mediaType)
	if PacketTypeAnnexB == pkt.packetType {
		return pkt.data
	}

	if pkt.dataAnnexB != nil {
		return pkt.dataAnnexB[:pkt.dataAnnexBSize]
	}

	bytes := make([]byte, len(pkt.data)+64)
	n := AVCC2AnnexB(bytes, pkt.data, nil)
	pkt.dataAnnexB = bytes
	pkt.dataAnnexBSize = n

	return pkt.dataAnnexB[:pkt.dataAnnexBSize]
}

func (pkt *avPacket) AVCCPacketData() []byte {
	Assert(AVMediaTypeVideo == pkt.mediaType)

	if PacketTypeAVCC == pkt.packetType {
		return pkt.data
	}

	if pkt.dataAVCC != nil {
		return pkt.dataAVCC[:pkt.dataAVCCSize]
	}

	bytes := make([]byte, len(pkt.data)+64)
	n := AnnexB2AVCC(bytes, pkt.data)
	pkt.dataAVCC = bytes
	pkt.dataAVCCSize = n

	return pkt.dataAVCC[:pkt.dataAVCCSize]
}

func (pkt *avPacket) Index() int {
	return pkt.index
}

func (pkt *avPacket) SetDuration(duration int64) {
	pkt.duration = duration
}

func (pkt *avPacket) Duration(timebase int) int64 {
	if pkt.timebase == timebase {
		return pkt.duration
	}

	return ConvertTs(pkt.duration, pkt.timebase, timebase)
}
