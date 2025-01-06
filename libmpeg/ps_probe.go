package libmpeg

import "fmt"

type PSProbeBuffer struct {
	deMuxer *PSDeMuxer
	buffer  []byte
	offset  int
	cap     int
}

func (p *PSProbeBuffer) Input(data []byte) error {
	length := len(data)
	size := p.offset + length

	var tmp = data
	if size > p.cap {
		return fmt.Errorf("probe buffer overflow: current length %d exceeds capacity %d", size, p.cap)
	} else if size != length {
		// 拷贝到缓冲区尾部
		copy(p.buffer[p.offset:], data)
		tmp = p.buffer[:size]
	}

	p.offset = size
	n, err := p.deMuxer.Input(tmp)
	if err != nil {
		return err
	} else if n > -1 {
		p.offset = size - n
		// 拷贝未解析完的剩余数据到缓冲区头部
		if p.offset > 0 {
			copy(p.buffer, tmp[n:])
		}
	}

	return nil
}

func NewProbeBuffer(deMuxer *PSDeMuxer, buffer []byte) *PSProbeBuffer {
	return &PSProbeBuffer{
		deMuxer: deMuxer,
		buffer:  buffer,
		offset:  0,
		cap:     cap(buffer),
	}
}
