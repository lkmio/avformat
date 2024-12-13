package librtmp

import (
	"encoding/binary"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
)

// https://en.wikipedia.org/wiki/Real-Time_Messaging_Protocol
// https://rtmp.veriskope.com/pdf/rtmp_specification_1.0.pdf
type ChunkType byte
type ChunkStreamID int
type MessageTypeID int
type MessageStreamID int
type UserControlMessageEvent uint16
type TransactionID int

/*
Chunk Format
Each Chunk consists of a header and Body. The header itself has
three parts:
+--------------+----------------+--------------------+--------------+
| Basic Header | Chunk Header | Extended Timestamp | Chunk Body |
+--------------+----------------+--------------------+--------------+
| |
|<------------------- Chunk Header ----------------->|
*/

/**
type 0
0 1 2 3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Timestamp |message length |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| message length (cont) |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| message stream id (cont) |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

const (
	ChunkType0 = ChunkType(0x00)
	ChunkType1 = ChunkType(0x01) //The message stream ID is not included
	ChunkType2 = ChunkType(0x02) //Neither the stream ID nor the message length is included;
	ChunkType3 = ChunkType(0x03) //basic header

	ChunkStreamIdNetwork = ChunkStreamID(2)
	ChunkStreamIdSystem  = ChunkStreamID(3)
	ChunkStreamIdAudio   = ChunkStreamID(4)
	ChunkStreamIdVideo   = ChunkStreamID(6)
	ChunkStreamIdSource  = ChunkStreamID(8)

	MessageTypeIDSetChunkSize               = MessageTypeID(1)
	MessageTypeIDAbortMessage               = MessageTypeID(2)
	MessageTypeIDAcknowledgement            = MessageTypeID(3)
	MessageTypeIDUserControlMessage         = MessageTypeID(4)
	MessageTypeIDWindowAcknowledgementSize  = MessageTypeID(5)
	MessageTypeIDSetPeerBandWith            = MessageTypeID(6)
	MessageTypeIDAudio                      = MessageTypeID(8)
	MessageTypeIDVideo                      = MessageTypeID(9)
	MessageTypeIDDataAMF0                   = MessageTypeID(18) // MessageTypeIDDataAMF0 MessageTypeIDDataAMF3 metadata:creation time, duration, theme...
	MessageTypeIDDataAMF3                   = MessageTypeID(15)
	MessageTypeIDCommandAMF0                = MessageTypeID(20) // MessageTypeIDCommandAMF0 MessageTypeIDCommandAMF3  connect, createStream, publish, play, pause
	MessageTypeIDCommandAMF3                = MessageTypeID(17)
	MessageTypeIDSharedObjectAMF0           = MessageTypeID(19)
	MessageTypeIDSharedObjectAMF3           = MessageTypeID(16)
	MessageTypeIDAggregateMessage           = MessageTypeID(22)
	UserControlMessageEventStreamBegin      = UserControlMessageEvent(0x00)
	UserControlMessageEventStreamEOF        = UserControlMessageEvent(0x01)
	UserControlMessageEventStreamDry        = UserControlMessageEvent(0x02)
	UserControlMessageEventSetBufferLength  = UserControlMessageEvent(0x03)
	UserControlMessageEventStreamIsRecorded = UserControlMessageEvent(0x04)
	UserControlMessageEventPingRequest      = UserControlMessageEvent(0x06)
	UserControlMessageEventPingResponse     = UserControlMessageEvent(0x07)

	TransactionIDConnect      = TransactionID(1)
	TransactionIDCreateStream = TransactionID(2)
	TransactionIDPlay         = TransactionID(0)
	DefaultChunkSize          = 128
	ChunkSize                 = 60000
	WindowSize                = 2500000

	MessageResult        = "_result"
	MessageError         = "_error"
	MessageConnect       = "connect"
	MessageCall          = "call"
	MessageClose         = "close"
	MessageFcPublish     = "FCPublish"
	MessageReleaseStream = "releaseStream"
	MessageCreateStream  = "createStream"
	MessageStreamLength  = "getStreamLength"
	MessagePublish       = "publish"
	MessagePlay          = "play"
	MessagePlay2         = "play2"
	MessageDeleteStream  = "deleteStream"
	MessageReceiveAudio  = "receiveAudio"
	MessageReceiveVideo  = "receiveVideo"
	MessageSeek          = "seek"
	MessagePause         = "pause"
	MessageOnStatus      = "onStatus"
	MessageOnMetaData    = "onMetaData"
)

type Chunk struct {
	// basic header
	Type           ChunkType     // fmt 1-3bytes.低6位等于0,2字节;低6位等于1,3字节
	ChunkStreamID_ ChunkStreamID // currentChunk stream id. customized by users

	Timestamp uint32        // type0-绝对时间戳/type1和type2-与上一个chunk的差值
	Length    int           // message length
	TypeID    MessageTypeID // message type id
	StreamID  uint32        // message stream id. customized by users. LittleEndian

	Body []byte // 消息体
	Size int    // 消息体大小
}

func (h *Chunk) MarshalHeader(dst []byte) int {
	var index int
	index++

	dst[0] = byte(h.Type) << 6
	if h.ChunkStreamID_ <= 63 {
		dst[0] = dst[0] | byte(h.ChunkStreamID_)
	} else if h.ChunkStreamID_ <= 0xFF {
		dst[0] = dst[0] & 0xC0
		dst[1] = byte(h.ChunkStreamID_)
		index++
	} else if h.ChunkStreamID_ <= 0xFFFF {
		dst[0] = dst[0] & 0xC0
		dst[0] = dst[0] | 0x1
		binary.BigEndian.PutUint16(dst[1:], uint16(h.ChunkStreamID_))
		index += 2
	}

	if h.Type < ChunkType3 {
		if h.Timestamp >= 0xFFFFFF {
			libbufio.WriteUInt24(dst[index:], 0xFFFFFF)
		} else {
			libbufio.WriteUInt24(dst[index:], h.Timestamp)
		}
		index += 3
	}

	if h.Type < ChunkType2 {
		libbufio.WriteUInt24(dst[index:], uint32(h.Length))
		index += 4
		dst[index-1] = byte(h.TypeID)
	}

	if h.Type < ChunkType1 {
		binary.LittleEndian.PutUint32(dst[index:], h.StreamID)
		index += 4
	}

	if h.Timestamp >= 0xFFFFFF {
		binary.BigEndian.PutUint32(dst[index:], h.Timestamp)
		index += 4
	}

	return index
}

func (h *Chunk) Marshal(dst []byte, chunkSize int) int {
	utils.Assert(len(h.Body) >= h.Length)
	n := h.MarshalHeader(dst)
	var length = h.Length

	for length > 0 {
		if length != h.Length {
			dst[n] = byte((0x3 << 6) | h.ChunkStreamID_)
			n++
		}

		consume := libbufio.MinInt(length, chunkSize)
		offset := h.Length - length
		copy(dst[n:], h.Body[offset:offset+consume])
		length -= consume
		n += consume
	}

	return n
}

func (h *Chunk) WriteBody(dst, data []byte, chunkSize int, offset int) int {
	utils.Assert(chunkSize-offset > 0)
	length := len(data)
	var n int

	for length > 0 {
		var min int
		if offset > 0 {
			min = libbufio.MinInt(length, chunkSize-offset)
			offset = 0
		} else {
			min = libbufio.MinInt(length, chunkSize)
		}

		copy(dst[n:], data[:min])
		n += min

		length -= min
		data = data[min:]

		// 写一个ChunkType3用作分割
		if length > 0 {
			dst[n] = (0x3 << 6) | byte(h.ChunkStreamID_)
			n++

			if h.Timestamp >= 0xFFFFFF {
				binary.BigEndian.PutUint32(dst[n:], h.Timestamp)
				n += 4
			}
		}
	}

	return n
}

func (h *Chunk) Reset() {
	//h.ChunkStreamID_ = 0
	//如果当前包没有携带timestamp字段, 默认和前一包一致
	//h.Timestamp = 0
	//如果当前包没有携带length字段, 默认和前一包一致
	//h.Length = 0
	//如果当前包没有携带tid字段, 默认和前一包一致
	//h.TypeID = 0
	h.StreamID = 0
	h.Size = 0
}

func NewAudioChunk() Chunk {
	//	CHUNK_TYPE_0,CHUNK_STREAM_ID_AUDIO,ts,MESSAGE_TYPE_ID_AUDIO,0,Body,Size
	return Chunk{
		Type:           ChunkType0,
		ChunkStreamID_: ChunkStreamIdAudio,
		TypeID:         MessageTypeIDAudio,
	}
}

func NewVideoChunk() Chunk {
	return Chunk{
		Type:           ChunkType0,
		ChunkStreamID_: ChunkStreamIdVideo,
		TypeID:         MessageTypeIDVideo,
	}
}
