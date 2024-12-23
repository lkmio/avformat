package libmpeg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/lkmio/avformat/utils"
)

const (
	PacketStartCode       = 0x000001BA
	SystemHeaderStartCode = 0x000001BB
	PSMStartCode          = 0x000001BC
	ProgramEndCode        = 0x000001B9

	trickModeControlTypeFastForward = 0x0
	trickModeControlTypeSlowMotion  = 0x1
	trickModeControlTypeFreezeFrame = 0x2
	trickModeControlTypeFastReverse = 0x3
	trickModeControlTypeSlowReverse = 0x4

	//Reference from https://github.com/FFmpeg/FFmpeg/blob/master/libavformat/mpeg.h
	StreamTypeVideoMPEG1     = 0x01
	StreamTypeVideoMPEG2     = 0x02
	StreamTypeAudioMPEG1     = 0x03
	StreamTypeAudioMPEG2     = 0x04
	StreamTypePrivateSection = 0x05
	StreamTypePrivateData    = 0x06
	StreamTypeAudioAAC       = 0x0F
	StreamTypeAudioMpeg4AAC  = 0x11
	StreamTypeVideoMpeg4     = 0x10
	StreamTypeVideoH264      = 0x1B
	StreamTypeVideoHEVC      = 0x24
	StreamTypeVideoCAVS      = 0x42
	StreamTypeAudioAC3       = 0x81

	StreamTypeAudioG711A = 0x90
	StreamTypeAudioG711U = 0x91
)

var (
	streamTypes           map[int]int
	codecId2StreamTypeMap map[utils.AVCodecID]int
)

type StreamType int

func (s StreamType) isAudio() bool {
	return streamTypes[int(s)] == StreamIdAudio
}

func (s StreamType) isVideo() bool {
	return streamTypes[int(s)] == StreamIdVideo || streamTypes[int(s)] == StreamIdH624
}

func init() {
	streamTypes = map[int]int{
		StreamTypeVideoMPEG1:     StreamIdVideo,
		StreamTypeVideoMPEG2:     StreamIdVideo,
		StreamTypeAudioMPEG1:     StreamIdAudio,
		StreamTypeAudioMPEG2:     StreamIdAudio,
		StreamTypePrivateSection: StreamIdPrivateStream1,
		StreamTypePrivateData:    StreamIdPrivateStream1,
		StreamTypeAudioAAC:       StreamIdAudio,
		StreamTypeVideoMpeg4:     StreamIdVideo,
		StreamTypeVideoH264:      StreamIdVideo,
		StreamTypeVideoHEVC:      StreamIdVideo,
		StreamTypeVideoCAVS:      StreamIdVideo,
		StreamTypeAudioAC3:       StreamIdAudio,
	}

	codecId2StreamTypeMap = map[utils.AVCodecID]int{
		utils.AVCodecIdMP3: StreamTypeVideoMPEG1,
		utils.AVCodecIdAAC: StreamTypeAudioAAC,

		utils.AVCodecIdH264:  StreamTypeVideoH264,
		utils.AVCodecIdHEVC:  StreamTypeVideoHEVC,
		utils.AVCodecIdMPEG4: StreamTypeVideoMpeg4,
	}
}

type PacketHeader struct {
	mpeg2                         bool
	systemClockReferenceBase      int64  //33
	systemClockReferenceExtension uint16 //9
	programMuxRate                uint32 //22

	stuffing []byte
}

func (h *PacketHeader) ToBytes(dst []byte) int {
	binary.BigEndian.PutUint32(dst, PacketStartCode)
	//2bits 01
	dst[4] = 0x40
	//3bits [32..30]
	dst[4] = dst[4] | (byte(h.systemClockReferenceBase>>30) << 3)
	//1bit marker bit
	dst[4] = dst[4] | 0x4
	//15bits [29..15]
	//2bits 29 28
	dst[4] = dst[4] | byte(h.systemClockReferenceBase>>28&0x3)
	//8bits
	dst[5] = byte(h.systemClockReferenceBase >> 20)
	//5bits
	dst[6] = byte(h.systemClockReferenceBase >> 12 & 0xF8)
	dst[6] = dst[6] | 0x4
	//15bits [14:0]
	//2bits
	dst[6] = dst[6] | byte(h.systemClockReferenceBase>>13&0x3)
	dst[7] = byte(h.systemClockReferenceBase >> 5)
	//5bits
	dst[8] = byte(h.systemClockReferenceBase&0x1f) << 3
	dst[8] = dst[8] | 0x4

	dst[8] = dst[8] | byte(h.systemClockReferenceExtension>>7&0x3)
	dst[9] = byte(h.systemClockReferenceExtension) << 1
	//1bits mark bit
	dst[9] = dst[9] | 0x1

	dst[10] = byte(h.programMuxRate >> 14)
	dst[11] = byte(h.programMuxRate >> 6)
	dst[12] = byte(h.programMuxRate) << 2
	//2bits 2 mark bit
	dst[12] = dst[12] | 0x3

	//5bits reserved
	//3bits pack_stuffing_length
	dst[13] = 0xF8
	offset := 14
	if h.stuffing != nil {
		length := len(h.stuffing)
		dst[13] = dst[13] | byte(length)
		copy(dst[offset:], h.stuffing)
		offset += length
	}

	return offset
}

func readPackHeader(header *PacketHeader, src []byte) int {
	length := len(src)
	if length < 14 {
		return -1
	}
	header.mpeg2 = src[4]&0xC0 == 0
	//mpeg1 版本占用4bits 没有clockExtension reserved stuffingLength
	header.systemClockReferenceBase = int64(src[4]&0x38)<<27 | (int64(src[4]&0x3) << 28) | (int64(src[5]) << 20) | (int64(src[6]&0xF8) << 12) | (int64(src[6]&0x3) << 13) | (int64(src[7]) << 5) | (int64(src[8] & 0xF8 >> 3))

	header.systemClockReferenceExtension = uint16(src[8]&0x3) << 7
	header.systemClockReferenceExtension = header.systemClockReferenceExtension | uint16(src[9]>>1)

	header.programMuxRate = uint32(src[10]) << 14
	header.programMuxRate = header.programMuxRate | uint32(src[11])<<6
	header.programMuxRate = header.programMuxRate | uint32(src[12]>>2)

	l := 14 + int(src[13]&0x7)
	if l > length {
		return -1
	}

	header.stuffing = src[14:l]
	return l
}

//func (h *PacketHeader) SetStuffing(stuffing []byte) {
//	if len(stuffing) > 7 {
//		panic("Stuffing length is only 3 bits")
//	}
//	h.stuffing = stuffing
//}

func (h *PacketHeader) ToString() string {
	if h.stuffing == nil {
		return fmt.Sprintf("systemClockReferenceBase=%d\r\nsystemClockReferenceExtension=%d\r\nprogramMuxRate=%d\r\n", h.systemClockReferenceBase,
			h.systemClockReferenceExtension, h.programMuxRate)
	} else {
		return fmt.Sprintf("systemClockReferenceBase=%d\r\nsystemClockReferenceExtension=%d\r\nprogramMuxRate=%d\r\nstuffingLength=%d\r\nstuffing=%s\r\n", h.systemClockReferenceBase,
			h.systemClockReferenceExtension, h.programMuxRate, len(h.stuffing), hex.EncodeToString(h.stuffing))
	}
}

// streamHeader 3bytes.
type streamHeader struct {
	streamId byte
	//'11'
	bufferBoundScale byte   //1
	bufferSizeBound  uint16 //13
}

func (h *streamHeader) ToString() string {
	return fmt.Sprintf("streamId=%x\r\nbufferBoundScale=%d\r\nbufferSizeBound=%d\r\n", h.streamId, h.bufferBoundScale, h.bufferSizeBound)
}

func (h *streamHeader) ToBytes(data []byte) {
	data[0] = h.streamId
	data[1] = 0xc0
	data[1] = data[1] | (h.bufferBoundScale << 5)
	data[1] = data[1] | byte(h.bufferSizeBound&0x1F00>>8)
	data[2] = byte(h.bufferSizeBound & 0xFF)
}

// SystemHeader 系统头标记流的stream id: 00 00 00 `E0`, 00 00 00 `C0`
type SystemHeader struct {
	//6 bytes
	rateBound                 uint32 //22
	audioBound                byte   //6 [0,32]
	fixedFlag                 byte   //1
	cspsFlag                  byte   //1
	systemAudioLockFlag       byte   //1
	systemVideoLockFlag       byte   //1
	videoBound                byte   //5 [0,16]
	packetRateRestrictionFlag byte   //1

	streams []streamHeader
}

func (h *SystemHeader) findStream(id byte) (streamHeader, bool) {
	if h.streams == nil {
		return streamHeader{}, false
	}
	for _, s := range h.streams {
		if s.streamId == id {
			return s, true
		}
	}

	return streamHeader{}, false
}

func readSystemHeader(header *SystemHeader, src []byte) int {
	length := len(src)
	if length < 6 {
		return -1
	}

	totalLength := 6 + (int(src[4])<<8 | int(src[5]))
	if totalLength > length {
		return -1
	}

	header.rateBound = uint32(src[6]) & 0x7E << 15
	header.rateBound = header.rateBound | uint32(src[7])<<7
	header.rateBound = header.rateBound | uint32(src[8]>>1)

	header.audioBound = src[9] >> 2
	header.fixedFlag = src[9] >> 1 & 0x1
	header.cspsFlag = src[9] & 0x1

	header.systemAudioLockFlag = src[10] >> 7
	header.systemVideoLockFlag = src[10] >> 6 & 0x1
	header.videoBound = src[10] & 0x1F
	header.packetRateRestrictionFlag = src[11] >> 7

	offset := 12
	for ; offset < totalLength && (src[offset]&0x80) == 0x80 && (totalLength-offset)%3 == 0; offset += 3 {
		if _, ok := header.findStream(src[offset]); ok {
			continue
		}

		sHeader := streamHeader{}
		sHeader.streamId = src[offset]
		sHeader.bufferBoundScale = src[offset+1] >> 5 & 0x1
		sHeader.bufferSizeBound = uint16(src[offset+1]&0x1F) << 8
		sHeader.bufferSizeBound = sHeader.bufferSizeBound | uint16(src[offset+2])
		header.streams = append(header.streams, sHeader)
	}

	return totalLength
}

func (h *SystemHeader) ToBytes(dst []byte) int {
	binary.BigEndian.PutUint32(dst, SystemHeaderStartCode)
	dst[6] = 0x80
	dst[6] = dst[6] | byte(h.rateBound>>15)
	dst[7] = byte(h.rateBound >> 7)
	dst[8] = byte(h.rateBound) << 1
	//mark bit
	dst[8] = dst[8] | 0x1
	dst[9] = h.audioBound << 2
	dst[9] = dst[9] | (h.fixedFlag << 1)
	dst[9] = dst[9] | h.cspsFlag

	dst[10] = h.systemAudioLockFlag << 7
	dst[10] = dst[10] | (h.systemVideoLockFlag << 6)
	dst[10] = dst[10] | 0x20
	dst[10] = dst[10] | h.videoBound

	dst[11] = h.packetRateRestrictionFlag << 7
	dst[11] = dst[11] | 0x7F

	offset := 12
	for i, s := range h.streams {
		s.ToBytes(dst[offset:])
		offset += (i + 1) * 3
	}

	binary.BigEndian.PutUint16(dst[4:], uint16(offset-6))
	return offset
}

func (h *SystemHeader) ToString() string {
	sprintf := fmt.Sprintf(
		"rateBound=%d\r\n"+
			"audioBound=%d\r\n"+
			"fixedFlag=%d\r\n"+
			"cspsFlag=%d\r\n"+
			"systemAudioLockFlag=%d\r\n"+
			"systemVideoLockFlag=%d\r\n"+
			"videoBound=%d\r\n"+
			"packetRateRestrictionFlag=%d\r\n",
		h.rateBound,
		h.audioBound,
		h.fixedFlag,
		h.cspsFlag,
		h.systemAudioLockFlag,
		h.systemVideoLockFlag,
		h.videoBound,
		h.packetRateRestrictionFlag,
	)

	sprintf += "streams=[\r\n"
	for _, stream := range h.streams {
		sprintf += stream.ToString()
	}
	sprintf += "]\r\n"

	return sprintf
}

type ElementaryStream struct {
	streamType byte //2-34. 0x5 disable
	streamId   byte
	info       []byte
}

func (e ElementaryStream) ToString() string {
	if e.info == nil {
		return fmt.Sprintf("StreamType=%x\r\nstreamId=%x\r\n", e.streamType, e.streamId)
	} else {
		return fmt.Sprintf("StreamType=%x\r\nstreamId=%x\r\ninfo=%s\r\n", e.streamType, e.streamId, hex.EncodeToString(e.info))
	}
}

// ProgramStreamMap 映射头标记流的编码器信息
type ProgramStreamMap struct {
	streamId             byte
	currentNextIndicator byte //1 bit
	version              byte //5 bits
	info                 []byte
	elementaryStreams    []ElementaryStream
	crc32                uint32
}

func (h *ProgramStreamMap) findElementaryStream(streamId byte) (ElementaryStream, bool) {
	if h.elementaryStreams == nil {
		return ElementaryStream{}, false
	}

	for _, element := range h.elementaryStreams {
		if element.streamId == streamId {
			return element, true
		}
	}

	return ElementaryStream{}, false
}

func (h *ProgramStreamMap) ToString() string {
	var info string
	if h.info != nil {
		info = hex.EncodeToString(h.info)
	}

	var elements string
	if h.elementaryStreams != nil {
		for _, element := range h.elementaryStreams {
			elements += element.ToString()
		}
	}

	return fmt.Sprintf("streamId=%x\r\ncurrentNextIndicator=%d\r\nversion=%d\r\ninfo=%s\r\nelements=[\r\n%s]\r\ncrc32=%d\r\n",
		h.streamId, h.currentNextIndicator, h.version, info, elements, h.crc32)
}

func readProgramStreamMap(header *ProgramStreamMap, src []byte) (int, error) {
	length := len(src)
	if length < 16 {
		return -1, nil
	}
	totalLength := 6 + int(binary.BigEndian.Uint16(src[4:]))
	if totalLength > length {
		return -1, nil
	}

	header.streamId = src[3]
	header.currentNextIndicator = src[6] >> 7
	header.version = src[6] & 0x1F

	infoLength := int(binary.BigEndian.Uint16(src[8:]))
	offset := 10
	if infoLength > 0 {
		// +2 reserved elementary_stream_map_length
		if 10+2+infoLength > totalLength-4 {
			return -1, fmt.Errorf("invalid data:%s", hex.EncodeToString(src))
		}

		offset += infoLength
		header.info = src[10:offset]
	}

	elementaryLength := int(binary.BigEndian.Uint16(src[offset:]))
	offset += 2
	if offset+elementaryLength > totalLength-4 {
		return -1, fmt.Errorf("invalid data:%s", hex.EncodeToString(src))
	}

	for i := offset; i < offset+elementaryLength; i += 4 {
		eInfoLength := int(binary.BigEndian.Uint16(src[i+2:]))

		if _, ok := header.findElementaryStream(src[i+1]); !ok {
			element := ElementaryStream{}
			element.streamType = src[i]
			element.streamId = src[i+1]

			if eInfoLength > 0 {
				//if i+4+eInfoLength > offset+elementaryLength {
				if i+4+eInfoLength > totalLength-4 {
					return 0, fmt.Errorf("invalid data:%s", hex.EncodeToString(src))
				}
				element.info = src[i+4 : i+4+eInfoLength]
			}

			header.elementaryStreams = append(header.elementaryStreams, element)
		}

		i += eInfoLength
	}

	header.crc32 = binary.BigEndian.Uint32(src[totalLength-4:])
	return totalLength, nil
}

func (h *ProgramStreamMap) ToBytes(dst []byte) int {
	binary.BigEndian.PutUint32(dst, PSMStartCode)
	//current_next_indicator
	dst[6] = 0x80
	//reserved
	dst[6] = dst[6] | (0x3 << 5)
	//program_stream_map_version
	dst[6] = dst[6] | 0x1
	//reserved
	dst[7] = 0xFE
	//mark bit
	dst[7] = dst[7] | 0x1

	offset := 10
	if h.info != nil {
		length := len(h.info)
		copy(dst[offset:], h.info)
		binary.BigEndian.PutUint16(dst[8:], uint16(length))
		offset += length
	} else {
		binary.BigEndian.PutUint16(dst[8:], 0)
	}
	//elementary length
	offset += 2
	temp := offset
	for _, elementaryStream := range h.elementaryStreams {
		dst[offset] = elementaryStream.streamType
		offset++
		dst[offset] = elementaryStream.streamId
		offset += 3
		if elementaryStream.info != nil {
			length := len(elementaryStream.info)
			copy(dst[offset:], elementaryStream.info)
			binary.BigEndian.PutUint16(dst[offset-2:], uint16(length))
			offset += length
		} else {
			binary.BigEndian.PutUint16(dst[offset-2:], 0)
		}
	}

	elementaryLength := offset - temp
	binary.BigEndian.PutUint16(dst[temp-2:], uint16(elementaryLength))

	crc32 := utils.CalculateCrcMpeg2(dst[:offset])
	binary.BigEndian.PutUint32(dst[offset:], crc32)

	offset += 4
	binary.BigEndian.PutUint16(dst[4:], uint16(offset-6))

	return offset
}
