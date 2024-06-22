package librtmp

import (
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
)

type Parser struct {
	state ParserState

	csidSize     int
	headerOffset int
	extended     bool

	chunks       []*Chunk
	currentChunk Chunk
	audioChunk   Chunk
	videoChunk   Chunk

	partPacketCB    func(chunk *Chunk, data []byte)
	localChunkSize  int //(我方)发送消息的chunk大小 默认128
	remoteChunkSize int //(对方)解析消息的chunk大小 默认128
}

func NewParser() *Parser {
	return &Parser{localChunkSize: DefaultChunkSize, remoteChunkSize: DefaultChunkSize}
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

			p.currentChunk.csid = 0
			if data[i]&0x3F == 0 {
				p.csidSize = 1
			} else if data[i]&0x3F == 1 {
				p.csidSize = 2
			} else {
				p.csidSize = 0
				p.currentChunk.csid = ChunkStreamID(data[i] & 0x3F)
			}

			p.currentChunk.type_ = t
			p.state = ParserStateBasicHeader
			i++
			break

		case ParserStateBasicHeader:
			for ; p.csidSize > 0 && i < length; i++ {
				p.currentChunk.csid <<= 8
				p.currentChunk.csid |= ChunkStreamID(data[i])
				p.csidSize--
			}

			if p.csidSize == 0 {
				if p.currentChunk.type_ < ChunkType3 {
					p.state = ParserStateTimestamp
					p.headerOffset = 0
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateTimestamp:
			for ; p.headerOffset < 3 && i < length; i++ {
				p.currentChunk.Timestamp <<= 8
				p.currentChunk.Timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 3 {
				p.currentChunk.Timestamp &= 0xFFFFFF
				p.extended = p.currentChunk.Timestamp == 0xFFFFFF
				if p.currentChunk.type_ < ChunkType2 {
					p.state = ParserStateMessageLength
				} else if p.extended {
					p.state = ParserStateExtendedTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateMessageLength:
			for ; p.headerOffset < 6 && i < length; i++ {
				p.currentChunk.Length <<= 8
				p.currentChunk.Length |= int(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 6 {
				//24位有效
				p.currentChunk.Length &= 0x00FFFFFF
				p.state = ParserStateStreamType
			}
			break

		case ParserStateStreamType:
			p.currentChunk.tid = MessageTypeID(data[i])
			i++
			p.headerOffset++
			if p.currentChunk.type_ == ChunkType0 {
				p.state = ParserStateStreamId
			} else if p.extended {
				p.state = ParserStateExtendedTimestamp
			} else {
				p.state = ParserStatePayload
			}
			break

		case ParserStateStreamId:
			for ; p.headerOffset < 11 && i < length; i++ {
				p.currentChunk.sid <<= 8
				p.currentChunk.sid |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 11 {
				if p.extended {
					p.state = ParserStateExtendedTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateExtendedTimestamp:
			for ; p.headerOffset < 15 && i < length; i++ {
				p.currentChunk.Timestamp <<= 8
				p.currentChunk.Timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 15 {
				p.state = ParserStatePayload
			}
			break

		case ParserStatePayload:
			rest := length - i

			var chunk *Chunk

			if MessageTypeIDAudio == p.currentChunk.tid || (p.currentChunk.tid == 0 && ChunkStreamIdAudio == p.currentChunk.csid) {
				chunk = &p.audioChunk
				chunk.tid = MessageTypeIDAudio
			} else if MessageTypeIDVideo == p.currentChunk.tid || (p.currentChunk.tid == 0 && ChunkStreamIdVideo == p.currentChunk.csid) {
				chunk = &p.videoChunk
				chunk.tid = MessageTypeIDVideo
			} else {
				//根据tid匹配chunk
				if p.currentChunk.tid != 0 {
					for _, c := range p.chunks {
						if c.tid == p.currentChunk.tid {
							chunk = c
							break
						}
					}
				}

				//根据csid匹配chunk
				if chunk == nil {
					for _, c := range p.chunks {
						if c.csid == p.currentChunk.csid {
							chunk = c
							break
						}
					}
				}

				if chunk == nil {
					chunk = &Chunk{}
					*chunk = p.currentChunk
					p.chunks = append(p.chunks, chunk)
				}
			}

			if p.currentChunk.Timestamp > 0 && p.currentChunk.Timestamp != chunk.Timestamp {
				chunk.Timestamp = p.currentChunk.Timestamp
			}
			if p.currentChunk.Length > 0 && p.currentChunk.Length != chunk.Length {
				chunk.Length = p.currentChunk.Length
			}

			chunk.type_ = p.currentChunk.type_
			chunk.sid = p.currentChunk.sid

			if chunk.Length == 0 {
				//p.Reset()
				//continue
				return nil, -1, fmt.Errorf("bad message. the length of an rtmp message cannot be zero")
			}

			need := chunk.Length - chunk.size
			consume := libbufio.MinInt(need, p.remoteChunkSize-(chunk.size%p.remoteChunkSize))
			consume = libbufio.MinInt(consume, rest)

			//实际推流中发现, obs音视频chunk包的csid都为4, 所以csid不能关联chunk. ffmpeg推流, 可能前面携带tid, 后面的包不携带tid, 所以此时要参考csid.
			if (&p.audioChunk == chunk || &p.videoChunk == chunk) && p.partPacketCB != nil {
				p.partPacketCB(chunk, data[i:i+consume])
			} else {
				if len(chunk.data) < chunk.Length {
					bytes := make([]byte, chunk.Length+1024)
					copy(bytes, chunk.data[:chunk.size])
					chunk.data = bytes
				}

				copy(chunk.data[chunk.size:], data[i:i+consume])
			}

			chunk.size += consume
			i += consume

			if chunk.size >= chunk.Length {
				p.state = ParserStateInit
				return chunk, i, nil
			} else if chunk.size%p.remoteChunkSize == 0 {
				p.state = ParserStateInit
			}
			break
		}
	}

	return nil, -1, nil
}

func (p *Parser) Reset() {
	p.currentChunk.Reset()
	p.currentChunk.Length = 0
	p.currentChunk.csid = 0
	p.currentChunk.tid = 0

	p.state = ParserStateInit
	p.csidSize = 0
	p.headerOffset = 0
	p.extended = false
}
