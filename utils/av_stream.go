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

	M4VCExtraData() ([]byte, error)

	AnnexBExtraData() ([]byte, error)
}

func NewAVStream(type_ AVMediaType, index int, codecId AVCodecID, extra []byte, extraType ExtraType) AVStream {
	stream := &avStream{type_: type_, index: index, codecId: codecId, data: extra, extraType: extraType}
	return stream
}

type avStream struct {
	type_ AVMediaType

	index int

	codecId AVCodecID

	data []byte

	extraAnnexB     []byte
	extraAnnexBSize int

	extraM4CV     []byte
	extraM4CVSize int

	extraType ExtraType
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
	return a.data
}

func (a *avStream) SetExtraData(data []byte) {
	a.data = data
}

func (a *avStream) M4VCExtraData() ([]byte, error) {
	//ast.TypeAssertExpr{}

	return nil, nil
}

func (a *avStream) AnnexBExtraData() ([]byte, error) {
	if ExtraTypeAnnexB == a.extraType {
		return a.data, nil
	}
	if a.extraAnnexB != nil {
		return a.extraAnnexB[:a.extraAnnexBSize], nil
	}

	b, err := M4VCExtraDataToAnnexB(a.data)
	if err != nil {
		return nil, err
	}

	a.extraAnnexB = b
	a.extraAnnexBSize = len(b)

	return a.extraAnnexB[:a.extraAnnexBSize], nil
}
