package librtp

type H265Muxer struct {
	muxer
	//FU头数据
	//indicator+header2个字节
	fuHeader []byte
}

func NewH265Muxer(payload int, seq int, ssrc uint32) Muxer {
	m := &H265Muxer{}
	m.init(payload, seq, ssrc)
	m.fuHeader = make([]byte, 3)
	m.maxPayloadSize -= 3
	return m
}

// Input 输入不包含start code的单个NalU
func (m *H265Muxer) Input(data []byte, timestamp uint32) {
	type_ := data[0] >> 1 & 0x3F
	length := len(data)

	//小于RTP(MTU)负载大小的NalU, 单一打包
	//小于RTP(MTU)负载大小的NalU, 分片使用FU-A打包
	if length < m.maxPayloadSize {
		m.muxer.Input(data, timestamp)
	} else {
		//0                   1                   2                   3
		//0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		//+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//|    PayloadHdr (Type=49)       |   FU header   | DONL (cond)   |
		//+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
		//| DONL (cond)   |                                               |
		//|-+-+-+-+-+-+-+-+                                               |
		//|                         FU payload                            |
		//|                                                               |
		//|                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//|                               :...OPTIONAL RTP padding        |
		//+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//Figure 9: The Structure of an FU

		//H265的NALU Header
		//+---------------+---------------+
		//|0|1|2|3|4|5|6|7|0|1|2|3|4|5|6|7|
		//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//|F|   Type    |  LayerId  | TID |
		//	+-------------+-----------------+
		//FU分包
		m.fuHeader[0] = 49 << 1
		//F+LayerId+TID
		m.fuHeader[1] = 1 /*data[0]&0x81 | data[1]*/

		splitPayloadData(data[2:], m.maxPayloadSize, func(payload []byte, start, end bool) {
			if start {
				m.fuHeader[2] = 0x80
			} else if !end {
				m.fuHeader[2] = 0x00
			} else {
				m.fuHeader[2] = 0x40
			}

			m.fuHeader[2] |= type_
			m.mux(timestamp, end, m.fuHeader, payload)
		})
	}
}
