package librtp

import (
	"github.com/lkmio/avformat/libbufio"
)

type Muxer interface {
	compose(bytes []byte, data ...[]byte) int

	mux(dst []byte, ts uint32, end bool, data ...[]byte) int

	init(payload int, seq int, ssrc uint32)

	Input(data []byte, timestamp uint32, alloc func() []byte, write func([]byte)) int

	GetHeader() *Header
}

type muxer struct {
	header         *Header
	headerLength   int
	maxPayloadSize int  // rtp包最大负载数据大小
	enableMark     bool // 是否使用mark标记位
}

func (m *muxer) compose(bytes []byte, data ...[]byte) int {
	// 内部拷贝, 内存拷贝消耗低于用户态和内核态的交互
	// seq内部自行递增
	n := m.header.Marshal(bytes)
	for _, data_ := range data {
		copy(bytes[n:], data_)
		n += len(data_)
	}

	return n
}

func (m *muxer) mux(dst []byte, ts uint32, end bool, data ...[]byte) int {
	m.header.Timestamp = ts
	//Set mark for the last packet.
	if m.enableMark {
		if end {
			m.header.m = 1
		} else {
			m.header.m = 0
		}
	}

	return m.compose(dst, data...)
}

// 按照指定大小分割负载数据, 如果start和end都为true, 说明len(data) < size
func splitPayloadData(data []byte, size int, callback func(data []byte, start, end bool)) int {
	length := len(data)
	tmp := length
	var count int

	for tmp > 0 {
		n := libbufio.MinInt(tmp, size)
		callback(data[length-tmp:length-tmp+n], tmp == length, tmp == n)
		tmp -= n

		count++
	}

	return count
}

func (m *muxer) Input(data []byte, timestamp uint32, alloc func() []byte, write func([]byte)) int {
	m.header.Timestamp = timestamp

	return splitPayloadData(data, m.maxPayloadSize, func(payload []byte, start, end bool) {
		bytes := alloc()
		n := m.mux(bytes, timestamp, end, payload)
		write(bytes[:n])
	})
}

func (m *muxer) init(payload int, seq int, ssrc uint32) {
	header := NewHeader(payload)
	header.Seq = uint16(seq)
	header.SSRC = ssrc
	m.header = header
	m.headerLength = FixedHeaderLength
	m.maxPayloadSize = PacketMaxSize - m.headerLength
	m.enableMark = true
}

func (m *muxer) GetHeader() *Header {
	return m.header
}

func NewMuxer(payload int, seq int, ssrc uint32) Muxer {
	m := &muxer{}
	m.init(payload, seq, ssrc)
	return m
}
