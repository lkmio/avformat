package libmp4

import "github.com/lkmio/avformat/utils"

type MetaData interface {
	MediaType() utils.AVMediaType
	CodeId() utils.AVCodecID
	ExtraData() []byte
	setMediaType(mediaType utils.AVMediaType)
	setCodeId(id utils.AVCodecID)
	setExtraData(data []byte)
}

type metaDataImpl struct {
	mediaType utils.AVMediaType
	codecId   utils.AVCodecID
	extra     []byte
}

func (m *metaDataImpl) MediaType() utils.AVMediaType {
	return m.mediaType
}

func (m *metaDataImpl) CodeId() utils.AVCodecID {
	return m.codecId
}

func (m *metaDataImpl) ExtraData() []byte {
	return m.extra
}

func (m *metaDataImpl) setMediaType(mediaType utils.AVMediaType) {
	m.mediaType = mediaType
}

func (m *metaDataImpl) setCodeId(id utils.AVCodecID) {
	m.codecId = id
}

func (m *metaDataImpl) setExtraData(data []byte) {
	m.extra = data
}

type VideoMetaData struct {
	metaDataImpl
	width  int
	height int

	lengthSize int
}

func (v *VideoMetaData) Width() int {
	return v.width
}

func (v *VideoMetaData) Height() int {
	return v.height
}

func (v *VideoMetaData) SetLengthSize(lengthSize int) {
	v.lengthSize = lengthSize
}

func (v *VideoMetaData) LengthSize() int {
	return v.lengthSize
}

type AudioMetaData struct {
	metaDataImpl
	sampleRate   int
	sampleBit    int
	channelCount int
}

func (a *AudioMetaData) MediaType() utils.AVMediaType {
	return a.mediaType
}

func (a *AudioMetaData) CodeId() utils.AVCodecID {
	return a.codecId
}

func (a *AudioMetaData) setMediaType(mediaType utils.AVMediaType) {
	a.mediaType = mediaType
}

func (a *AudioMetaData) setCodeId(id utils.AVCodecID) {
	a.codecId = id
}

func (a *AudioMetaData) SampleRate() int {
	return a.sampleRate
}

func (a *AudioMetaData) SampleBit() int {
	return a.sampleBit
}

func (a *AudioMetaData) ChannelCount() int {
	return a.channelCount
}

type SubTitleMetaData struct {
	metaDataImpl
}
