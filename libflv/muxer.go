package libflv

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat/utils"
	"time"
)

type Muxer interface {
	AddVideoTrack(id utils.AVCodecID)

	AddAudioTrack(id utils.AVCodecID, soundRate, soundType, soundSize int)

	WriteHeader(data []byte) int

	Input(dst []byte, mediaType utils.AVMediaType, pktSize int, dts, pts int64, key, header_ bool) int

	WriteTag(dst []byte, mediaType utils.AVMediaType, dataSize, timestamp uint32) int

	WriteAudioData(dst []byte, header bool) int

	WriteVideoData(dst []byte, ct uint32, key, header bool) int

	// AddProperty 添加元数据
	AddProperty(name string, value interface{}) error
}

type muxer struct {
	existAudio bool
	existVideo bool

	videoCodecId VideoCodecId
	soundFormat  SoundFormat
	soundRate    SoundRate
	soundType    byte //0-mono/1-stereo. for Nellymoser always:0, For AAC always:1
	soundSize    byte //0-8位深/1-16位深
	preSize      uint32

	metaData *AMF0Object
}

func NewMuxer() Muxer {
	m := &muxer{
		soundSize: 1,
		metaData:  &AMF0Object{},
	}

	m.metaData.AddStringProperty("creationtime", time.Now().Format("2006-01-02 15:04:05"))
	return m
}

func (m *muxer) AddVideoTrack(id utils.AVCodecID) {
	utils.Assert(!m.existVideo)

	if utils.AVCodecIdH264 == id {
		m.videoCodecId = VideoCodeIdH264
	} else {
		utils.Assert(false)
	}

	m.existVideo = true
	m.metaData.AddNumberProperty("videocodecid", float64(m.videoCodecId))
}

func (m *muxer) AddAudioTrack(id utils.AVCodecID, soundRate, soundType, soundSize int) {
	utils.Assert(!m.existAudio)

	if utils.AVCodecIdAAC == id {
		m.soundFormat = SoundFormatAAC
		m.soundRate = SoundRate44000HZ
		m.soundType = 1
	} else if utils.AVCodecIdPCMALAW == id {
		m.soundFormat = SoundFormatG711A
		m.soundRate = SoundRate44000HZ
		m.soundType = 0
	} else if utils.AVCodecIdPCMALAW == id {
		m.soundFormat = SoundFormatG711B
		m.soundRate = SoundRate44000HZ
		m.soundType = 0
	} else if utils.AVCodecIdMP3 == id {
		m.soundFormat = SoundFormatMP3
		m.soundRate = SoundRate44000HZ
		m.soundType = 1
	} else {
		utils.Assert(false)
	}

	m.existAudio = true
	m.metaData.AddNumberProperty("audiocodecid", float64(m.soundFormat))
	m.metaData.AddNumberProperty("audiosamplerate", float64(m.soundRate))
}

func (m *muxer) WriteHeader(data []byte) int {
	//signature
	data[0] = 0x46
	data[1] = 0x4C
	data[2] = 0x56
	//version
	data[3] = 0x1
	//flags
	var flags byte
	if m.existAudio {
		flags |= 1 << 2
	}
	if m.existVideo {
		flags |= 1
	}

	data[4] = flags

	binary.BigEndian.PutUint32(data[5:], 0x9)
	amf0 := NewAMF0Writer()
	amf0.AddString("onMetaData")
	amf0.AddObject(m.metaData)
	//先写metadata
	n := amf0.ToBytes(data[9+15:])
	//再写tag
	m.writeTag(data[9:], TagTypeScriptDataObject, uint32(n), 0)
	return 9 + 15 + n
}

func (m *muxer) Input(dst []byte, mediaType utils.AVMediaType, pktSize int, dts, pts int64, key, header_ bool) int {
	if utils.AVMediaTypeAudio == mediaType {
		_ = dst[16]
		n := m.WriteTag(dst, mediaType, uint32(pktSize+2), uint32(dts))
		n += m.WriteAudioData(dst[n:], header_)
		return n
	} else if utils.AVMediaTypeVideo == mediaType {
		_ = dst[19]
		n := m.WriteTag(dst, mediaType, uint32(pktSize+5), uint32(dts))
		n += m.WriteVideoData(dst[n:], uint32(pts-dts), key, header_)
		return n
	}

	panic("")
}

func (m *muxer) writeTag(dst []byte, tag TagType, dataSize, timestamp uint32) int {
	binary.BigEndian.PutUint32(dst, m.preSize)
	dst[4] = byte(tag)
	utils.WriteUInt24(dst[5:], dataSize)
	utils.WriteUInt24(dst[8:], timestamp&0xFFFFFF)
	dst[11] = byte(timestamp >> 24)
	utils.WriteUInt24(dst[12:], 0)

	m.preSize = 11 + dataSize
	return 15
}

func (m *muxer) WriteTag(dst []byte, mediaType utils.AVMediaType, dataSize, timestamp uint32) int {
	var tag TagType

	if utils.AVMediaTypeAudio == mediaType {
		tag = TagTypeAudioData
	} else if utils.AVMediaTypeVideo == mediaType {
		tag = TagTypeVideoData
	}

	return m.writeTag(dst, tag, dataSize, timestamp)
}

func (m *muxer) WriteAudioData(dst []byte, header bool) int {
	_ = dst[1]
	dst[0] = byte(m.soundFormat)<<4 | byte(m.soundRate)<<2 | m.soundSize<<1 | m.soundType
	if header {
		dst[1] = 0
	} else {
		dst[1] = 1
	}

	return 2
}

func (m *muxer) WriteVideoData(dst []byte, ct uint32, key, header bool) int {
	_ = dst[4]
	var frameType byte
	if header || key {
		frameType = 1
	} else {
		frameType = 0
	}

	dst[0] = frameType<<4 | byte(m.videoCodecId)
	if header {
		dst[1] = 0
	} else {
		dst[1] = 1
	}
	utils.WriteUInt24(dst[2:], ct)

	return 5
}

func (m *muxer) AddProperty(name string, value interface{}) error {
	if s, ok := value.(string); ok {
		m.metaData.AddStringProperty(name, s)
	} else if s, ok := value.(float64); ok {
		m.metaData.AddNumberProperty(name, s)
	} else if s, ok := value.(int); ok {
		m.metaData.AddNumberProperty(name, float64(s))
	} else if s, ok := value.(uint); ok {
		m.metaData.AddNumberProperty(name, float64(s))
	} else {
		return fmt.Errorf("unknow property %v", value)
	}

	return nil
}
