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

	CodecParameters() CodecData
}

type avStream struct {
	type_ AVMediaType

	index int

	codecId AVCodecID

	data []byte

	codecParameters CodecData
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

func (a *avStream) CodecParameters() CodecData {
	return a.codecParameters
}

type AudioStream struct {
	avStream

	SampleRate int
	Channels   int
}

func NewAVStream(type_ AVMediaType, index int, codecId AVCodecID, extra []byte, config CodecData) AVStream {
	return &avStream{type_: type_, index: index, codecId: codecId, data: extra, codecParameters: config}
}

func NewAudioStream(type_ AVMediaType, index int, codecId AVCodecID, extra []byte, sampleRate, channels int) AVStream {
	return &AudioStream{avStream: avStream{type_: type_, index: index, codecId: codecId, data: extra}, SampleRate: sampleRate, Channels: channels}
}
