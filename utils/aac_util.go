package utils

import (
	"encoding/binary"
	"fmt"
)

type AudioObjectType int

const (
	AotNull = AudioObjectType(0)
	// Support?                Name
	AotAacMain       = AudioObjectType(1)  ///< Y                       Main
	AotAacLc         = AudioObjectType(2)  ///< Y                       Low Complexity
	AotAacSsr        = AudioObjectType(3)  ///< N (code in SoC repo)    Scalable Sample Rate
	AotAacLtp        = AudioObjectType(4)  ///< Y                       Long Term Prediction
	AotSbr           = AudioObjectType(5)  ///< Y                       Spectral Band Replication
	AotAacScalable   = AudioObjectType(6)  ///< N                       Scalable
	AotTwinvq        = AudioObjectType(7)  ///< N                       Twin Vector Quantizer
	AotCelp          = AudioObjectType(8)  ///< N                       Code Excited Linear Prediction
	AotHvxc          = AudioObjectType(9)  ///< N                       Harmonic Vector eXcitation Coding
	AotTtsi          = AudioObjectType(12) ///< N                       Text-To-Speech Interface
	AotMainsynth     = AudioObjectType(13) ///< N                       Main Synthesis
	AotWavesynth     = AudioObjectType(14) ///< N                       Wavetable Synthesis
	AotMidi          = AudioObjectType(15) ///< N                       General MIDI
	AotSafx          = AudioObjectType(16) ///< N                       Algorithmic Synthesis and Audio Effects
	AotErAacLc       = AudioObjectType(17) ///< N                       Error Resilient Low Complexity
	AotErAacLtp      = AudioObjectType(19) ///< N                       Error Resilient Long Term Prediction
	AotErAacScalable = AudioObjectType(20) ///< N                       Error Resilient Scalable
	AotErTwinvq      = AudioObjectType(21) ///< N                       Error Resilient Twin Vector Quantizer
	AotErBsac        = AudioObjectType(22) ///< N                       Error Resilient Bit-Sliced Arithmetic Coding
	AotErAacLd       = AudioObjectType(23) ///< N                       Error Resilient Low Delay
	AotErCelp        = AudioObjectType(24) ///< N                       Error Resilient Code Excited Linear Prediction
	AotErHvxc        = AudioObjectType(25) ///< N                       Error Resilient Harmonic Vector eXcitation Coding
	AotErHiln        = AudioObjectType(26) ///< N                       Error Resilient Harmonic and Individual Lines plus Noise
	AotErParam       = AudioObjectType(27) ///< N                       Error Resilient Parametric
	AotSsc           = AudioObjectType(28) ///< N                       SinuSoidal Coding
	AotPs            = AudioObjectType(29) ///< N                       Parametric Stereo
	AotSurround      = AudioObjectType(30) ///< N                       MPEG Surround
	AotEscape        = AudioObjectType(31) ///< Y                       Escape Value
	AotL1            = AudioObjectType(32) ///< Y                       Layer 1
	AotL2            = AudioObjectType(33) ///< Y                       Layer 2
	AotL3            = AudioObjectType(34) ///< Y                       Layer 3
	AotDst           = AudioObjectType(35) ///< N                       Direct Stream Transfer
	AotAls           = AudioObjectType(36) ///< Y                       Audio LosslesS
	AotSls           = AudioObjectType(37) ///< N                       Scalable LosslesS
	AotSlsNonCore    = AudioObjectType(38) ///< N                       Scalable LosslesS (non core)
	AotErAacEld      = AudioObjectType(39) ///< N                       Error Resilient Enhanced Low Delay
	AotSmrSimple     = AudioObjectType(40) ///< N                       Symbolic Music Representation Simple
	AotSmrMain       = AudioObjectType(41) ///< N                       Symbolic Music Representation Main
	//2019 ?
	AotUsacNosbr  = AudioObjectType(42) ///< N                       Unified Speech and Audio Coding (no SBR)
	AotSaoc       = AudioObjectType(43) ///< N                       Spatial Audio Object Coding
	AotLdSurround = AudioObjectType(44) ///< N                       Low Delay MPEG Surround
	AotUsac       = AudioObjectType(45) ///< N                       Unified Speech and Audio Coding

	DefaultAACFrameLength = 1024
)

var (
	audioSamplingRates = map[int]int{
		0:  96000,
		1:  88200,
		2:  64000,
		3:  48000,
		4:  44100,
		5:  32000,
		6:  24000,
		7:  22050,
		8:  16000,
		9:  12000,
		10: 11025,
		11: 8000,
		12: 7350,
		13: -1,
		14: -1,
		15: -1,
	}
	mpeg4AudioChannels = [14]int{
		0,
		1, // mono (1/0)
		2, // stereo (2/0)
		3, // 3/0
		4, // 3/1
		5, // 3/2
		6, // 3/2.1
		8, // 5/2.1
		0,
		0,
		0,
		7,  // 3/3.1
		8,  // 3/2/2.1
		24, // 3/3/3 - 5/2/3 - 3/0/0.2
	}
)

type MPEG4AudioConfig struct {
	ObjectType    int //5bits
	SamplingIndex int //4bits
	SampleRate    int
	ChanConfig    int //4bits
	Channels      int

	sbr              int ///< -1 implicit, 1 presence
	extObjectType    int
	extSamplingIndex int
	extSampleRate    int
	extChanConfig    int
	ps               int ///< -1 implicit, 1 presence
	frameLengthShort int
}

type ADtsHeader uint64

func (a ADtsHeader) SyncWord() int {
	return int(a) >> 44 & 0xFFF
}

// ID 0-mpeg4/1-mpeg2
func (a ADtsHeader) ID() int {
	return int(a) >> 43 & 0x1
}

func (a ADtsHeader) Layer() int {
	return int(a) >> 41 & 0x3
}

// ProtectionAbsent 1-header length=7/0-header length=9(多2个字节crc校验码)
func (a ADtsHeader) ProtectionAbsent() int {
	return int(a) >> 40 & 0x1
}

// Profile Aot减1
func (a ADtsHeader) Profile() int {
	return int(a) >> 38 & 0x3
}

func (a ADtsHeader) Frequency() int {
	return int(a) >> 34 & 0xF
}

func (a ADtsHeader) PrivateBit() int {
	return int(a) >> 33 & 0x1
}

func (a ADtsHeader) Channel() int {
	return int(a) >> 30 & 0x7
}

func (a ADtsHeader) Original() int {
	return int(a) >> 29 & 0x1
}

func (a ADtsHeader) Home() int {
	return int(a) >> 28 & 0x1
}

func (a ADtsHeader) CopyrightBit() int {
	return int(a) >> 27 & 0x1
}

func (a ADtsHeader) CopyrightStart() int {
	return int(a) >> 26 & 0x1
}

func (a ADtsHeader) FrameLength() int {
	return int(a) >> 13 & 0x1FFF
}

func (a ADtsHeader) Fullness() int {
	return int(a) >> 2 & 0x7FF
}

func (a ADtsHeader) Blocks() int {
	return int(a) & 0x3
}

func GetSampleRateFromFrequency(index int) (int, bool) {
	i, ok := audioSamplingRates[index]
	return i, ok
}

func GetSampleRateIndex(rate int) int {
	return audioSamplingRates[rate]
}

func SetADtsHeader(header []byte, mpegId, aot, index, channelConfig, size int) {
	header[0] = 0xFF
	//id MPEG Version:0-MPEG4/1-MPEG2
	header[1] = 0xF0 | (byte(mpegId) & 0x1 << 3)
	//layer
	header[1] = header[1] | 0x0<<1
	//protection_absent
	header[1] = header[1] | 0x1
	//profile object type 2bits
	header[2] = byte(aot) & 0x3 << 6
	//sample rate index 4bits
	header[2] = header[2] | (byte(index) & 0xF << 2)
	//private bit
	header[2] = header[2] | byte(0x0<<1)
	//3 bits
	header[2] = header[2] | (byte(channelConfig) >> 2 & 0x1)
	header[3] = byte(channelConfig) << 6
	header[3] = header[3] | 0x0<<5
	header[3] = header[3] | 0x0<<4

	//adts_variable_header
	header[3] = header[3] | 0x0<<3
	header[3] = header[3] | 0x0<<2
	//aac_frame_length 13bits
	header[3] = header[3] | byte(size>>11&0x3)
	header[4] = byte(size >> 3)
	header[5] = byte(size&0x7) << 5
	//adts_buffer_fullness 0x7FF 11bits
	header[5] = header[5] | 0x1F
	header[6] = 0xFC // the last 2 bytes of byte belong to number_of_raw_data_blocks_in_frame
}

func ReadADtsFixedHeader(data []byte) (ADtsHeader, error) {
	if 0xFFF != binary.BigEndian.Uint16(data)>>4 {
		return 0, fmt.Errorf("not find syncword")
	}

	var header uint64
	header = uint64(int(data[0])<<16|int(data[1])<<8|int(data[2])) << 32
	header |= uint64(binary.BigEndian.Uint32(data[3:]))

	if ADtsHeader(header).ProtectionAbsent() == 0 {
	}

	return ADtsHeader(header), nil
}

func ADtsHeader2MpegAudioConfigData(header ADtsHeader) ([]byte, error) {
	bytes := make([]byte, 2)
	profile := header.Profile()
	bytes[0] = (uint8(profile) + 1) & 0x1F << 3
	frequency := header.Frequency()

	bytes[0] |= uint8(frequency) >> 1 & 0x7
	bytes[1] = uint8(frequency) & 0x1 << 7
	bytes[1] |= uint8(header.Channel()) & 0xF << 3

	return bytes, nil
}

func ParseMpeg4AudioConfig(data []byte) (*MPEG4AudioConfig, error) {
	//audio specific config
	config := &MPEG4AudioConfig{}
	config.ObjectType = int(data[0] >> 3)
	if AudioObjectType(config.ObjectType) == AotEscape {
		//config.ObjectType = 32+bits6
	}
	config.SamplingIndex = int((data[0] & 7 << 1) | data[1]>>7)
	config.SampleRate = audioSamplingRates[(config.SamplingIndex)]
	config.ChanConfig = int(data[1] >> 3 & 0xF)
	if config.ChanConfig >= len(mpeg4AudioChannels) {
		return nil, fmt.Errorf("invalid data")
	}

	config.Channels = mpeg4AudioChannels[config.ChanConfig]
	return config, nil
	//config.sbr = -1
	//config.ps = -1
	//if config.ObjectType == AotSbr || (config.ObjectType == AotPs && )
}

func ComputeAACFrameDuration(sampleRate int) float32 {
	return float32(sampleRate) / float32(DefaultAACFrameLength)
}
