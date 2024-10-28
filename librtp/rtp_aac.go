package librtp

// AACMuxer 默认90KHZ
// 暂时只实现[RFC3640]3.3.6. High Bit-rate AAC的封装
type AACMuxer struct {
	muxer
	auHeader []byte
}

func (m *AACMuxer) Input(data []byte, timestamp uint32, alloc func() []byte, write func([]byte)) int {
	return splitPayloadData(data, m.maxPayloadSize, func(payload []byte, start, end bool) {
		m.auHeader[2] = byte(len(payload) >> 5)
		m.auHeader[3] = byte(len(payload) & 0x1F << 3)
		bytes := alloc()
		n := m.mux(bytes, timestamp, end, m.auHeader, payload)
		write(bytes[:n])
	})
}

func NewAACMuxer(payload int, seq int, ssrc uint32) Muxer {
	m := &AACMuxer{}
	m.init(payload, seq, ssrc)
	m.enableMark = true
	m.auHeader = make([]byte, 4)
	m.maxPayloadSize -= 4
	m.auHeader[0] = 0x00
	m.auHeader[1] = 0x10
	return m
}
