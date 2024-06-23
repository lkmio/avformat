package librtmp

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
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
Each chunk consists of a header and data. The header itself has
three parts:
+--------------+----------------+--------------------+--------------+
| Basic Header | Chunk Header | Extended Timestamp | Chunk Data |
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
	//basic header
	type_ ChunkType     //fmt 1-3bytes.低6位等于0,2字节;低6位等于1,3字节
	csid  ChunkStreamID //chunk stream id. customized by users

	Timestamp uint32
	//表明的chunk长度
	Length int           //message length
	tid    MessageTypeID //message type id
	sid    uint32        //message stream id. customized by users. LittleEndian

	//body
	data []byte
	//实际接受到的chunk大小
	size int
}

func NewAudioChunk() Chunk {
	//	CHUNK_TYPE_0,CHUNK_STREAM_ID_AUDIO,ts,MESSAGE_TYPE_ID_AUDIO,0,data,size
	return Chunk{
		type_: ChunkType0,
		csid:  ChunkStreamIdAudio,
		tid:   MessageTypeIDAudio,
	}
}

func NewVideoChunk() Chunk {
	return Chunk{
		type_: ChunkType0,
		csid:  ChunkStreamIdVideo,
		tid:   MessageTypeIDVideo,
	}
}

func (h *Chunk) ToBytes(dst []byte) int {
	var index int
	index++

	dst[0] = byte(h.type_) << 6
	if h.csid <= 63 {
		dst[0] = dst[0] | byte(h.csid)
	} else if h.csid <= 0xFF {
		dst[0] = dst[0] & 0xC0
		dst[1] = byte(h.csid)
		index++
	} else if h.csid <= 0xFFFF {
		dst[0] = dst[0] & 0xC0
		dst[0] = dst[0] | 0x1
		binary.BigEndian.PutUint16(dst[1:], uint16(h.csid))
		index += 2
	}

	if h.type_ < ChunkType3 {
		if h.Timestamp >= 0xFFFFFF {
			libbufio.WriteUInt24(dst[index:], 0xFFFFFF)
		} else {
			libbufio.WriteUInt24(dst[index:], uint32(h.Timestamp))
		}
		index += 3
	}

	if h.type_ < ChunkType2 {
		libbufio.WriteUInt24(dst[index:], uint32(h.Length))
		index += 4
		dst[index-1] = byte(h.tid)
	}

	if h.type_ < ChunkType1 {
		binary.LittleEndian.PutUint32(dst[index:], uint32(h.sid))
		index += 4
	}

	if h.Timestamp >= 0xFFFFFF {
		binary.BigEndian.PutUint32(dst[index:], uint32(h.Timestamp))
		index += 4
	}

	return index
}

func (h *Chunk) ToBytes2(data []byte, chunkSize int) int {
	utils.Assert(len(h.data) >= h.Length)
	n := h.ToBytes(data)
	var length = h.Length

	for length > 0 {
		if length != h.Length {
			data[n] = byte((0x3 << 6) | h.csid)
			n++
		}

		consume := libbufio.MinInt(length, chunkSize)
		offset := h.Length - length
		copy(data[n:], h.data[offset:offset+consume])
		length -= consume
		n += consume
	}

	return n
}

func (h *Chunk) WriteData(dst, data []byte, chunkSize int, offset int) int {
	length := len(data)
	first := true
	var n int
	for length > 0 {
		var min int
		if first {
			min = libbufio.MinInt(length, chunkSize-offset)
			first = false
		} else {
			min = libbufio.MinInt(length, chunkSize)
		}

		copy(dst[n:], data[:min])
		n += min

		length -= min
		data = data[min:]

		//写一个ChunkType3用作分割
		if length > 0 {
			dst[n] = (0x3 << 6) | byte(h.csid)
			n++
		}
	}

	return n
}

func readBasicHeader(src []byte) (ChunkType, ChunkStreamID, int, error) {
	t := ChunkType(src[0] >> 6)
	if t > 0x3 {
		return t, 0, 0, fmt.Errorf("unknow chunk type:%d", t)
	}

	switch src[0] & 0x3F {
	case 0:
		//64-(64+255)
		return t, ChunkStreamID(64 + int(src[1])), 2, nil
	case 1:
		//64-(65535+64)
		return t, ChunkStreamID(64 + int(binary.BigEndian.Uint16(src[1:]))), 3, nil
	//case 2:
	default:
		//1bytes
		return t, ChunkStreamID(src[0] & 0x3F), 1, nil
	}
}

func (h *Chunk) Reset() {
	//h.csid = 0
	//如果当前包没有携带timestamp字段, 默认和前一包一致
	//h.Timestamp = 0
	//如果当前包没有携带length字段, 默认和前一包一致
	//h.Length = 0
	//如果当前包没有携带tid字段, 默认和前一包一致
	//h.tid = 0
	h.sid = 0
	h.size = 0
}

func readChunkHeader(src []byte) (Chunk, int, error) {
	t, csid, i, err := readBasicHeader(src)
	if err != nil {
		return Chunk{}, 0, err
	}

	header := Chunk{
		type_: t,
		csid:  csid,
	}

	if header.type_ < ChunkType3 {
		header.Timestamp = uint32(libbufio.BytesToInt(src[i : i+3]))
		i += 3
	}

	if header.type_ < ChunkType2 {
		i += 3
		header.Length = libbufio.BytesToInt(src[i-3 : i])
		header.tid = MessageTypeID(src[i])
		i++
	}

	if header.type_ < ChunkType1 {
		header.sid = binary.LittleEndian.Uint32(src[i:])
		i += 4
	}

	if header.Timestamp == 0xFFFFFF {
		header.Timestamp = binary.BigEndian.Uint32(src[i:])
		i += 4
	}

	return header, i, nil
}
