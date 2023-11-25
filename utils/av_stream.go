package utils

type ExtraType int

const (
	ExtraTypeAnnexB = ExtraType(1)
	ExtraTypeM4VC   = ExtraType(2)
	ExtraTypeNONE   = ExtraType(3)
)

type AVStream interface {
	Index() int

	Type() AVMediaType

	CodecId() AVCodecID

	Extra() []byte

	SetExtraData(data []byte)

	M4VCExtraData() []byte

	AnnexBExtraData() []byte
}

func NewAVStream(type_ AVMediaType, index int, codecId AVCodecID, extra []byte, extraType ExtraType) AVStream {
	stream := &avStream{type_: type_, index: index, codecId: codecId, extra: extra, extraType: extraType}
	return stream
}

type avStream struct {
	type_ AVMediaType

	index int

	codecId AVCodecID

	extra []byte

	extraType ExtraType

	extraAnnexB ByteBuffer

	extraM4CV ByteBuffer
}

func (a *avStream) Index() int {
	return a.index
}

func (a *avStream) Type() AVMediaType {
	return a.type_
}

func (a *avStream) CodecId() AVCodecID {
	return a.codecId
}

func (a *avStream) Extra() []byte {
	return a.extra
}

func (a *avStream) SetExtraData(data []byte) {
	a.extra = data
}

func (a *avStream) M4VCExtraData() []byte {
	//ast.TypeAssertExpr{}

	return nil
}

func (a *avStream) AnnexBExtraData() []byte {

	return nil
}
