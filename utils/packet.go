package utils

type PacketType byte

const (
	PacketTypeAnnexB = PacketType(1)
	PacketTypeAVCC   = PacketType(2)
	PacketTypeNONE   = PacketType(3)
)

type AVPacket struct {
	data ByteBuffer
	pts  int64
	dts  int64
}

func (p *AVPacket) Pts() int64 {
	return p.pts
}

func (p *AVPacket) Dts() int64 {
	return p.dts
}

func (p *AVPacket) SetPts(pts int64) {
	p.pts = pts
}

func (p *AVPacket) SetDts(dts int64) {
	p.dts = dts
}

func (p *AVPacket) Data() ByteBuffer {
	return p.data
}

func (p *AVPacket) Write(data []byte) {
	p.data.Write(data)
}

func (p *AVPacket) Release() {
	p.data.Clear()
	p.pts = -1
	p.dts = -1
}

func NewPacket() *AVPacket {
	return &AVPacket{
		data: NewByteBuffer(),
		pts:  -1,
		dts:  -1,
	}
}

type AVPacket2 struct {
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

func NewAVPacket2(data []byte, dts, pts int64, key bool, packetType PacketType, mediaType AVMediaType, id AVCodecID) *AVPacket2 {
	return &AVPacket2{data: data, dts: dts, pts: pts, packetType: packetType, mediaType: mediaType, codecId: id}
}

func (pkt *AVPacket2) Dts() int64 {
	return pkt.dts
}

func (pkt *AVPacket2) Pts() int64 {
	return pkt.pts
}

func (pkt *AVPacket2) MediaType() AVMediaType {
	return pkt.mediaType
}

func (pkt *AVPacket2) Data() []byte {
	return pkt.data
}

func (pkt *AVPacket2) KeyFrame() bool {
	return pkt.key
}
