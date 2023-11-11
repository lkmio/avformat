package libmpeg

import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
)

type TSDeMuxer struct {
	pmt      []int
	pid      []int
	esPacket []byte
	esLength int
	packet   *utils.AVPacket

	lastPesPacket    *PESHeader
	currentPesPacket *PESHeader
	handler          deHandler
}

func NewTSDeMuxer(handler deHandler) *TSDeMuxer {
	return &TSDeMuxer{esPacket: make([]byte, 1024*1024*2), packet: utils.NewPacket(), currentPesPacket: NewPESPacket(), handler: handler}
}

func (t *TSDeMuxer) existPMT(id int) bool {
	for _, i := range t.pmt {
		if i == id {
			return true
		}
	}
	return false
}

func (t *TSDeMuxer) existPES(id int) bool {
	for _, i := range t.pid {
		if id == i {
			return true
		}
	}

	return false
}

func (t *TSDeMuxer) doRead(data []byte) error {
	h, i := readTSHeader(data)
	if h.pid == PSIPAT {
		pmt := readPAT(data[i:])
		for _, id := range pmt {
			if !t.existPMT(id) {
				t.pmt = append(t.pmt, id)
			}
		}

	} else if t.existPMT(h.pid) {
		pid := readPMT(data[i:])
		for _, id := range pid {
			if !t.existPES(id) {
				t.pid = append(t.pid, id)
			}
		}
	} else if t.existPES(h.pid) {
		if h.payloadUnitStartIndicator == 0x01 {
			n := readPESHeader(t.currentPesPacket, data[i:])
			if n == 0 {
				t.currentPesPacket.Reset()
				return fmt.Errorf("invaild data")
			}

			if t.lastPesPacket == nil {
				pesPacket := *t.currentPesPacket
				t.lastPesPacket = &pesPacket
			}
			//callback last pkt
			if t.currentPesPacket.streamId != t.lastPesPacket.streamId || t.currentPesPacket.pts != t.lastPesPacket.pts {
				if t.esLength > 0 {
					//t.callback()
					t.packet.Write(t.esPacket[:t.esLength])
					callbackES(t.lastPesPacket.streamId, t.lastPesPacket.streamId, t.packet, t.handler)
					t.packet.Release()
				}
				t.esLength = 0
				*t.lastPesPacket = *t.currentPesPacket
			}
			if t.currentPesPacket.ptsDtsFlags&0x3 != 0 {
				t.packet.SetPts(t.currentPesPacket.pts)
				t.packet.SetDts(t.currentPesPacket.dts)
			}

			i += n
		}

		if t.currentPesPacket == nil {
			return fmt.Errorf("invalid data")
		}
		copy(t.esPacket[t.esLength:], data[i:])
		t.esLength += len(data[i:])
		t.currentPesPacket.Reset()
	}
	return nil
}
