package librtp

import (
	"github.com/lkmio/avformat/libbufio"
)

type allocHandler func(params interface{}) []byte
type writeHandler func(data []byte, timestamp uint32, params interface{})

type Muxer interface {
	compose(bytes []byte, data ...[]byte) int

	mux(ts uint32, end bool, data ...[]byte)

	init(payload int, seq int, ssrc uint32)

	Input(data []byte, timestamp uint32)

	SetAllocHandler(handler allocHandler)

	SetWriteHandler(handler writeHandler)

	// SetParams 传递私有数据
	SetParams(params interface{})

	Close()
}

type muxer struct {
	header         *Header
	headerLength   int
	maxPayloadSize int //单个rtp包最大负载数据大小

	//是否使用mark标记位
	enableMark bool

	params interface{}

	allocHandler allocHandler
	writeHandler writeHandler
}

func (m *muxer) SetAllocHandler(handler allocHandler) {
	m.allocHandler = handler
}

func (m *muxer) SetWriteHandler(handler writeHandler) {
	m.writeHandler = handler
}

func (m *muxer) SetParams(params interface{}) {
	m.params = params
}

func (m *muxer) compose(bytes []byte, data ...[]byte) int {
	//内部拷贝, 内存拷贝消耗低于用户态和内核态的交互
	//seq内部自行递增
	n := m.header.toBytes(bytes)
	for _, data_ := range data {
		copy(bytes[n:], data_)
		n += len(data_)
	}

	return n
}

func (m *muxer) mux(ts uint32, end bool, data ...[]byte) {
	bytes := m.allocHandler(m.params)

	m.header.timestamp = ts
	//Set mark for the last packet.
	if m.enableMark {
		if end {
			m.header.m = 1
		} else {
			m.header.m = 0
		}
	}

	n := m.compose(bytes, data...)
	m.writeHandler(bytes[:n], ts, m.params)
}

// 按照指定大小分割负载数据, 如果start和end都为true, 说明len(data) < size
func splitPayloadData(data []byte, size int, callback func(data []byte, start, end bool)) {
	length := len(data)
	tmp := length
	for tmp > 0 {
		count := libbufio.MinInt(tmp, size)
		callback(data[length-tmp:length-tmp+count], tmp == length, tmp == count)
		tmp -= count
	}
}

func (m *muxer) Input(data []byte, timestamp uint32) {
	m.header.timestamp = timestamp
	splitPayloadData(data, m.maxPayloadSize, func(payload []byte, start, end bool) {
		m.mux(timestamp, end, payload)
	})
}

func (m *muxer) Close() {
	m.params = nil
	m.allocHandler = nil
	m.writeHandler = nil
}

func (m *muxer) init(payload int, seq int, ssrc uint32) {
	header := NewHeader()
	header.pt = byte(payload)
	header.seq = seq
	header.ssrc = ssrc
	m.header = header
	m.headerLength = FixedHeaderLength
	m.maxPayloadSize = PacketMaxSize - m.headerLength
	m.enableMark = true
}

func NewMuxer(payload int, seq int, ssrc uint32) Muxer {
	m := &muxer{}
	m.init(payload, seq, ssrc)
	return m
}
