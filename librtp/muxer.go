package librtp

import (
	"github.com/yangjiechina/avformat/utils"
)

type encodeHandler func(data []byte, timestamp uint32)

type Muxer struct {
	buffer       []byte
	header       *Header
	headerLength int
	payloadSize  int
	handler      encodeHandler
}

func (m *Muxer) Input(data []byte, timestamp uint32) {
	length, index := len(data), 0
	m.header.timestamp = timestamp
	for length > 0 {
		size := utils.MinInt(length, m.payloadSize)
		//Set mark for the last packet.
		if 0 == size-length {
			m.header.m = 1
		} else {
			m.header.m = 0
		}

		_ = m.header.toBytes(m.buffer)
		copy(m.buffer[m.headerLength:], data[index:index+size])
		length -= size
		index += size

		m.handler(m.buffer[:m.headerLength+size], timestamp)
	}
}

func NewMuxer(payload int, seq int, ssrc uint32, handler encodeHandler) *Muxer {
	header := NewHeader()
	header.pt = byte(payload)
	header.seq = seq
	header.ssrc = ssrc
	mux := &Muxer{buffer: make([]byte, 1500), header: header, handler: handler}
	mux.headerLength = FixedHeaderLength
	mux.payloadSize = PacketMaxSize - mux.headerLength
	return mux
}
