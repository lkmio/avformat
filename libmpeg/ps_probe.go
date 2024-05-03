package libmpeg

import "fmt"

type PSProbeBuffer struct {
	deMuxer *PSDeMuxer
	buffer  []byte
	offset  int
	cap     int
}

func (p PSProbeBuffer) Input(data []byte) error {
	length := len(data)
	if p.offset+length > p.cap {
		return fmt.Errorf("PS流解析缓冲区已满")
	}

	var n int
	var err error
	if p.offset == 0 {
		if n, err = p.deMuxer.Input(data); n > -1 {
			copy(p.buffer[:], data[n:])
			p.offset = length - n
		}
	} else {
		copy(p.buffer[p.offset:], data)
		p.offset += length

		if n, err = p.deMuxer.Input(p.buffer[:p.offset]); n > -1 {
			copy(p.buffer[:], p.buffer[n:p.offset])
			p.offset -= n
		}
	}

	return err
}

func NewProbeBuffer(deMuxer *PSDeMuxer, buffer []byte) *PSProbeBuffer {
	return &PSProbeBuffer{
		deMuxer: deMuxer,
		buffer:  buffer,
		offset:  0,
		cap:     cap(buffer),
	}
}
