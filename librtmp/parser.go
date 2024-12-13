package librtmp

import (
	"fmt"
	"github.com/lkmio/avformat/libbufio"
)

type ParserState byte

const (
	HandshakeStateUninitialized = HandshakeState(0) // after the client sends C0
	HandshakeStateVersionSent   = HandshakeState(1) // client waiting for S1
	HandshakeStateAckSent       = HandshakeState(2) // client waiting for S2
	HandshakeStateDone          = HandshakeState(3) // client receives S2
)

type Parser struct {
	state        ParserState
	headerOffset int
	extended     bool // 时间戳是否扩展

	chunks []*Chunk
	header Chunk

	onPartPacketCB     func(chunk *Chunk, data []byte) // 解析出部分音视频帧的回调
	localChunkMaxSize  int                             // 发送消息的chunk最大大小 默认128
	remoteChunkMaxSize int                             // 解析对方消息的chunk最大大小 默认128
}

func (p *Parser) ReadChunk(data []byte) (*Chunk, int, error) {
	length, i := len(data), 0
	for i < length {
		switch p.state {

		case ParserStateInit:
			t := ChunkType(data[i] >> 6)
			if t > ChunkType3 {
				return nil, -1, fmt.Errorf("unknow chunk type:%d", int(t))
			}

			p.header.ChunkStreamID_ = 0
			if data[i]&0x3F == 0 {
				p.headerOffset = 1
			} else if data[i]&0x3F == 1 {
				p.headerOffset = 2
			} else {
				p.headerOffset = 0
				p.header.ChunkStreamID_ = ChunkStreamID(data[i] & 0x3F)
			}

			p.header.Type = t
			p.state = ParserStateBasicHeader
			i++
			break

		case ParserStateBasicHeader:
			for ; p.headerOffset > 0 && i < length; i++ {
				p.header.ChunkStreamID_ <<= 8
				p.header.ChunkStreamID_ |= ChunkStreamID(data[i])
				p.headerOffset--
			}

			if p.headerOffset == 0 {
				if p.header.Type < ChunkType3 {
					p.state = ParserStateTimestamp
				} else if p.extended {
					p.state = ParserStateExtendedTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateTimestamp:
			for ; p.headerOffset < 3 && i < length; i++ {
				p.header.Timestamp <<= 8
				p.header.Timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 3 {
				p.headerOffset = 0

				p.header.Timestamp &= 0xFFFFFF
				p.extended = p.header.Timestamp == 0xFFFFFF
				if p.header.Type < ChunkType2 {
					p.state = ParserStateMessageLength
				} else if p.extended {
					p.state = ParserStateExtendedTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateMessageLength:
			for ; p.headerOffset < 3 && i < length; i++ {
				p.header.Length <<= 8
				p.header.Length |= int(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 3 {
				p.headerOffset = 0

				//24位有效
				p.header.Length &= 0x00FFFFFF
				p.state = ParserStateStreamType
			}
			break

		case ParserStateStreamType:
			p.header.TypeID = MessageTypeID(data[i])
			i++

			if p.header.Type == ChunkType0 {
				p.state = ParserStateStreamId
			} else if p.extended {
				p.state = ParserStateExtendedTimestamp
			} else {
				p.state = ParserStatePayload
			}
			break

		case ParserStateStreamId:
			for ; p.headerOffset < 4 && i < length; i++ {
				p.header.StreamID <<= 8
				p.header.StreamID |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 4 {
				p.headerOffset = 0

				if p.extended {
					p.state = ParserStateExtendedTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateExtendedTimestamp:
			for ; p.headerOffset < 4 && i < length; i++ {
				p.header.Timestamp <<= 8
				p.header.Timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 4 {
				p.headerOffset = 0

				p.state = ParserStatePayload
			}
			break

		case ParserStatePayload:
			// 根据Chunk Stream ID, Type ID查找或创建对应的Chunk
			var chunk *Chunk

			if p.header.TypeID != 0 {
				for _, c := range p.chunks {
					if c.TypeID == p.header.TypeID {
						chunk = c
						break
					}
				}
			}

			if chunk == nil && p.header.ChunkStreamID_ != 0 {
				for _, c := range p.chunks {
					if c.ChunkStreamID_ == p.header.ChunkStreamID_ {
						chunk = c
						break
					}
				}
			}

			if chunk == nil {
				chunk = &Chunk{}
				*chunk = p.header
				p.chunks = append(p.chunks, chunk)
			}

			if p.header.StreamID != 0 {
				chunk.StreamID = p.header.StreamID
			}

			if p.header.Length > 0 && p.header.Length != chunk.Length {
				chunk.Length = p.header.Length
			}

			// 以第一包的type为准
			if chunk.Size == 0 {
				chunk.Type = p.header.Type
			}

			// 时间戳为0, 认为和上一个包相同. 这是一种常见的节省空间的做法.
			if p.header.Timestamp > 0 {
				chunk.Timestamp = p.header.Timestamp
			}

			if p.header.TypeID > 0 {
				chunk.TypeID = p.header.TypeID
			}

			if chunk.Length == 0 {
				//p.Reset()
				//continue
				return nil, -1, fmt.Errorf("bad message. the length of an rtmp message cannot be zero")
			}

			// 计算能拷贝的有效长度
			rest := length - i
			need := chunk.Length - chunk.Size
			consume := libbufio.MinInt(need, p.remoteChunkMaxSize-(chunk.Size%p.remoteChunkMaxSize))
			consume = libbufio.MinInt(consume, rest)

			// 是否是音视频帧
			var av bool
			if MessageTypeIDAudio == p.header.TypeID || (ChunkStreamIdAudio == p.header.ChunkStreamID_ && p.header.TypeID == 0) {
				av = true
			} else if MessageTypeIDVideo == p.header.TypeID || (ChunkStreamIdVideo == p.header.ChunkStreamID_ && p.header.TypeID == 0) {
				av = true
			}

			if av && p.onPartPacketCB != nil {
				p.onPartPacketCB(chunk, data[i:i+consume])
			} else {
				if len(chunk.Body) < chunk.Length {
					bytes := make([]byte, chunk.Length+1024)
					copy(bytes, chunk.Body[:chunk.Size])
					chunk.Body = bytes
				}

				copy(chunk.Body[chunk.Size:], data[i:i+consume])
			}

			chunk.Size += consume
			i += consume

			if chunk.Size >= chunk.Length {
				p.state = ParserStateInit
				return chunk, i, nil
			} else if chunk.Size%p.remoteChunkMaxSize == 0 {
				p.state = ParserStateInit
			}
			break
		}
	}

	return nil, -1, nil
}

func (p *Parser) Reset() {
	p.header = Chunk{}
	p.state = ParserStateInit
	p.headerOffset = 0
	p.extended = false
}

func NewParser() *Parser {
	return &Parser{localChunkMaxSize: DefaultChunkSize, remoteChunkMaxSize: DefaultChunkSize}
}
