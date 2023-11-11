package librtmp

import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
)

type Parser struct {
	state     ParserState
	chunkType ChunkType
	csid      ChunkStreamID
	csidSize  int
	//headerSize        int
	headerOffset int
	extended     bool

	chunks map[ChunkStreamID]*Chunk
	chunk  *Chunk

	//chunk大小 默认128
	chunkSize int
}

func NewParser() *Parser {
	return &Parser{chunkSize: DefaultChunkSize, chunks: make(map[ChunkStreamID]*Chunk, 10)}
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

			if data[i]&0x3F == 0 {
				p.csidSize = 1
			} else if data[i]&0x3F == 1 {
				p.csidSize = 2
			} else {
				p.csidSize = 0
				p.csid = ChunkStreamID(data[i] & 0x3F)
			}

			p.chunkType = t
			//p.headerSize = headerSize[p.type_]
			p.state = ParserStateBasicHeader
			i++
			break

		case ParserStateBasicHeader:
			for ; p.csidSize > 0 && i < length; i++ {
				p.csid <<= 8
				p.csid |= ChunkStreamID(data[i])
				p.csidSize--
			}

			if p.csidSize == 0 {
				chunk := p.chunks[p.csid]
				//message := p.findMessage(p.csid)
				if chunk == nil {
					chunk = &Chunk{type_: p.chunkType, csid: p.csid}
					p.chunks[p.csid] = chunk
				}

				p.chunk = chunk
				if p.chunkType < ChunkType3 {
					p.state = ParserStateTimestamp
				} else {
					p.state = ParserStatePayload
				}
			}
			break

		case ParserStateTimestamp:
			for ; p.headerOffset < 3 && i < length; i++ {
				p.chunk.timestamp <<= 8
				p.chunk.timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 3 {
				p.extended = p.chunk.timestamp == 0xFFFFFF
				if p.chunkType < ChunkType2 {
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
				p.chunk.Length <<= 8
				p.chunk.Length |= int(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 6 {
				//24位有效
				p.chunk.Length &= 0x00FFFFFF
				p.state = ParserStateStreamType
			}
			break

		case ParserStateStreamType:
			p.chunk.tid = MessageTypeID(data[i])
			i++
			p.headerOffset++
			if p.chunkType == ChunkType0 {
				p.state = ParserStateStreamId
			} else if p.extended {
				p.state = ParserStateExtendedTimestamp
			} else {
				p.state = ParserStatePayload
			}
			break

		case ParserStateStreamId:
			for ; p.headerOffset < 11 && i < length; i++ {
				p.chunk.sid <<= 8
				p.chunk.sid |= int(data[i])
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
				p.chunk.timestamp <<= 8
				p.chunk.timestamp |= uint32(data[i])
				p.headerOffset++
			}

			if p.headerOffset == 15 {
				p.state = ParserStatePayload
			}
			break

		case ParserStatePayload:
			rest := length - i

			if p.chunk.Length == 0 {
				println(fmt.Printf("bad message. the length of an rtmp message cannot be zero."))
				break
			}

			need := p.chunk.Length - p.chunk.size
			consume := utils.MinInt(need, p.chunkSize-(p.chunk.size%p.chunkSize))
			consume = utils.MinInt(consume, rest)

			if len(p.chunk.data) < p.chunk.Length {
				bytes := make([]byte, p.chunk.Length+1024)
				copy(bytes, p.chunk.data)
				p.chunk.data = bytes
			}

			copy(p.chunk.data[p.chunk.size:], data[i:i+consume])
			p.chunk.size += consume

			i += consume
			if p.chunk.size >= p.chunk.Length {

				return p.chunk, i, nil
			} else if p.chunk.size%p.chunkSize == 0 {
				p.state = ParserStateInit
			}

			break
		}
	}

	return nil, -1, nil
}

func (p *Parser) Reset() {
	p.chunk.Reset()
	p.state = ParserStateInit
	p.csidSize = 0
	p.headerOffset = 0
	p.chunkType = ChunkType0
	p.csid = 0
	p.extended = false
}
