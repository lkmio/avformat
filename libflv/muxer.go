package libflv

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
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

	ComputeVideoDataSize(ct uint32) int

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
	} else if utils.AVCodecIdH265 == id {
		m.videoCodecId = VideoCodeIdHEVC
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
	n := amf0.ToBytes(data[9+TagHeaderSize:])
	//再写tag
	m.writeTag(data[9:], TagTypeScriptDataObject, uint32(n), 0)
	return 9 + TagHeaderSize + n
}

func (m *muxer) Input(dst []byte, mediaType utils.AVMediaType, pktSize int, dts, pts int64, key, header_ bool) int {
	if utils.AVMediaTypeAudio == mediaType {
		_ = dst[16]
		n := m.WriteAudioData(dst[TagHeaderSize:], header_)
		n += m.WriteTag(dst, mediaType, uint32(pktSize+n), uint32(dts))
		return n
	} else if utils.AVMediaTypeVideo == mediaType {
		_ = dst[19]
		n := m.WriteVideoData(dst[TagHeaderSize:], uint32(pts-dts), key, header_)
		n += m.WriteTag(dst, mediaType, uint32(pktSize+n), uint32(dts))
		return n
	}

	panic("")
}

func (m *muxer) writeTag(dst []byte, tag TagType, dataSize, timestamp uint32) int {
	binary.BigEndian.PutUint32(dst, m.preSize)
	dst[4] = byte(tag)
	libbufio.WriteUInt24(dst[5:], dataSize)
	libbufio.WriteUInt24(dst[8:], timestamp&0xFFFFFF)
	dst[11] = byte(timestamp >> 24)
	libbufio.WriteUInt24(dst[12:], 0)

	m.preSize = 11 + dataSize
	return TagHeaderSize
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

func (m *muxer) ComputeVideoDataSize(ct uint32) int {
	if ct > 0 {
		return 8
	}

	return 5
}

func (m *muxer) WriteVideoData(dst []byte, ct uint32, key, header bool) int {
	_ = dst[4]
	var frameType byte
	if header || key {
		frameType = 1
	} else {
		frameType = 0
	}

	n := 1
	setCt := true
	codecId := byte(m.videoCodecId)
	if VideoCodeIdHEVC == m.videoCodecId {
		setCt = ct != 0
		frameType |= 0b1000

		if header {
			codecId = byte(PacketTypeSequenceStart)
		} else {
			if setCt {
				codecId = byte(PacketTypeCodedFrames)
			} else {
				codecId = byte(PacketTypeCodedFramesX)
			}
		}

		binary.BigEndian.PutUint32(dst[n:], uint32(m.videoCodecId))
		n += 4
	} else {
		if header {
			dst[n] = 0
		} else {
			dst[n] = 1
		}

		n++
	}

	dst[0] = frameType<<4 | codecId

	if setCt {
		libbufio.WriteUInt24(dst[n:], ct)
		n += 3
	}

	return n
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
