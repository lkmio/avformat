package librtp

import (
	"encoding/binary"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
)

const (
	VERSION           = 2
	FixedHeaderLength = 12
	PacketMaxSize     = 1458
	PayloadMaxSize    = 0
	PayloadMinSize    = 261
)

type Header struct {
	v         byte //2
	p         byte //1
	x         byte //1
	cc        byte //4
	m         byte //1
	pt        byte //7
	seq       int
	timestamp uint32
	ssrc      uint32

	csrc             []uint32
	extensionProfile uint16
	extensionLength  uint16
	extensions       []uint32
}

func NewHeader() *Header {
	return &Header{
		v: VERSION,
	}
}

func (h *Header) toBytes(dst []byte) int {
	dst[0] = h.v << 6
	dst[0] = dst[0] | (h.p << 5)
	dst[0] = dst[0] | (h.x << 4)
	dst[0] = dst[0] | h.cc
	dst[1] = h.m << 7
	dst[1] = dst[1] | (h.pt & 0x7F)

	libbufio.WriteWORD(dst[2:], uint16(h.seq))
	libbufio.WriteDWORD(dst[4:], h.timestamp)
	libbufio.WriteDWORD(dst[8:], h.ssrc)

	//csrc
	offset := FixedHeaderLength
	if h.cc > 0 {
		for i, v := range h.csrc {
			offset += i * 4
			libbufio.WriteDWORD(dst[offset:], v)
		}
	}

	//extension
	if h.p > 0 {
		libbufio.WriteWORD(dst[offset:], h.extensionProfile)
		libbufio.WriteWORD(dst[offset+2:], h.extensionLength)
		offset += 4
		for i, v := range h.extensions {
			offset += i * 4
			libbufio.WriteDWORD(dst[offset:], v)
		}
	}

	h.seq = (h.seq + 1) & 0xFFFF
	return offset
}

func (h *Header) Length() int {
	length := FixedHeaderLength
	if h.cc > 0 {
		length += len(h.csrc) * 4
	}
	if h.p > 0 {
		length += 4
		length += len(h.extensions) * 4
	}
	return length
}

func (h *Header) SetCSRCList(list []uint32) {
	if len(list) > 15 {
		panic("the CSRC length only has 4 bits.")
	}
	h.cc = byte(len(list))
	h.csrc = list
}

func (h *Header) SetExtensions(profile uint16, extensions []uint32) {
	h.x = 1
	h.extensionProfile = profile
	h.extensionLength = uint16(len(extensions))
	h.extensions = extensions
}

func (h *Header) Padding() bool {
	return h.p == 1
}

func (h *Header) Extension() bool {
	return h.x == 1
}

func RollbackSeq(header []byte, nextSeq int) {
	utils.Assert(nextSeq < 65536)
	seq := nextSeq - 1
	if seq < 0 {
		seq += 65536

	}

	binary.BigEndian.PutUint16(header[2:], uint16(seq))
}
