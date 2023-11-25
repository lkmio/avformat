package libflv

import (
	"encoding/binary"
	"github.com/yangjiechina/avformat/utils"
)

type Muxer struct {
	existAudio bool
	existVideo bool

	videoCodecId VideoCodecId
	soundFormat  SoundFormat
	soundRate    SoundRate
	soundType    byte
	soundSize    byte
	preSize      uint32
}

func NewMuxer(audioCodecId, videoCodecId utils.AVCodecID, soundRate, soundType, soundSize int) *Muxer {
	m := &Muxer{
		existAudio: utils.AVCodecIdNONE != audioCodecId,
		existVideo: utils.AVCodecIdNONE != videoCodecId,
	}

	if m.existAudio {
		if utils.AVCodecIdAAC == audioCodecId {
			m.soundFormat = SoundFormatAAC
			m.soundRate = SoundRate44000HZ
			m.soundType = 1
		} else {
			utils.Assert(false)
		}
	}

	if m.existVideo {
		if utils.AVCodecIdH264 == videoCodecId {
			m.videoCodecId = VideoCodeIdH264
		} else {
			utils.Assert(false)
		}
	}

	return m
}

func (m *Muxer) WriteHeader(data []byte) int {
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

	return 9
}

func (m *Muxer) Input(dst []byte, mediaType utils.AVMediaType, pktSize int, dts, pts int64, key, header_ bool) int {
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

func (m *Muxer) WriteTag(dst []byte, mediaType utils.AVMediaType, dataSize, timestamp uint32) int {
	binary.BigEndian.PutUint32(dst, m.preSize)

	if utils.AVMediaTypeAudio == mediaType {
		dst[4] = byte(TagTypeAudioData)
	} else if utils.AVMediaTypeVideo == mediaType {
		dst[4] = byte(TagTypeVideoData)
	}

	utils.WriteUInt24(dst[5:], dataSize)
	utils.WriteUInt24(dst[8:], timestamp&0xFFFFFF)
	dst[11] = byte(timestamp >> 24)
	utils.WriteUInt24(dst[12:], 0)

	m.preSize = dataSize
	return 15
}

func (m *Muxer) WriteAudioData(dst []byte, header bool) int {
	_ = dst[1]
	dst[0] = byte(m.soundFormat)<<4 | byte(m.soundRate) | m.soundSize | m.soundType
	if header {
		dst[1] = 0
	} else {
		dst[1] = 1
	}

	return 2
}

func (m *Muxer) WriteVideoData(dst []byte, ct uint32, key, header bool) int {
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
