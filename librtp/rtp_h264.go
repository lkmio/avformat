package librtp

type H264Muxer struct {
	muxer

	// FU头数据
	// indicator+header2个字节
	fuHeader []byte
}

// Input 输入不包含start code的单个NalU
func (m *H264Muxer) Input(data []byte, timestamp uint32, alloc func() []byte, write func([]byte)) int {
	type_ := data[0] & 0x1F
	length := len(data)

	// 小于RTP(MTU)负载大小的NalU, 单一打包
	// 小于RTP(MTU)负载大小的NalU, 分片使用FU-A打包
	if length < m.maxPayloadSize {
		return m.muxer.Input(data, timestamp, alloc, write)
	} else {
		// 取原本的F和NRI
		m.fuHeader[0] = data[0] & 0xE0
		// FU-A 分片单元
		m.fuHeader[0] |= 28
		// S/E/R/TYPE
		m.fuHeader[1] = 0

		return splitPayloadData(data[1:], m.maxPayloadSize, func(payload []byte, start, end bool) {
			if start {
				m.fuHeader[1] = 0x80
			} else if !end {
				m.fuHeader[1] = 0x00
			} else {
				m.fuHeader[1] = 0x40
			}

			m.fuHeader[1] |= type_

			bytes := alloc()
			n := m.mux(bytes, timestamp, end, m.fuHeader, payload)
			write(bytes[:n])
		})
	}
}

func NewH264Muxer(payload int, seq int, ssrc uint32) Muxer {
	m := &H264Muxer{}
	m.init(payload, seq, ssrc)
	m.fuHeader = make([]byte, 2)
	m.maxPayloadSize -= 2
	return m
}
