package transport

import (
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
)

// FixedLengthFrameDecoder 固定长度解码器
type FixedLengthFrameDecoder struct {
	Data []byte
	Size int //已经读取数据长度
	cap  int
	cb   func([]byte)
}

func NewFixedLengthFrameDecoder(frameLength int, cb func([]byte)) *FixedLengthFrameDecoder {
	utils.Assert(frameLength > 0)

	return &FixedLengthFrameDecoder{Data: make([]byte, frameLength), Size: 0, cap: frameLength, cb: cb}
}

func (d *FixedLengthFrameDecoder) Input(data []byte) error {
	i, length := 0, len(data)

	remain := length - i

	for need := d.cap - d.Size; remain >= need; need = d.cap - d.Size {
		if d.Size == 0 {
			d.cb(data[i : i+d.cap])
		} else {
			copy(d.Data, data[i:i+need])
			d.cb(d.Data)
			i += need
			d.Size = 0
		}

		remain = length - i
	}

	copy(d.Data[d.Size:], data[i:])
	d.Size += remain

	return nil
}

// LengthFieldFrameDecoder 帧长解码器
type LengthFieldFrameDecoder struct {
	Data  []byte
	Size  int //已经读取数据长度
	total int //总长度
	cap   int

	cb func([]byte)

	fieldLength int //几个字节描述数据包长
}

func NewLengthFieldFrameDecoder(frameLength, fieldLength int, cb func([]byte)) *LengthFieldFrameDecoder {
	utils.Assert(frameLength > 0)
	utils.Assert(fieldLength > 0 && fieldLength < 5)

	frameLength += fieldLength

	return &LengthFieldFrameDecoder{Data: make([]byte, frameLength), Size: 0, cap: frameLength, cb: cb, fieldLength: fieldLength}
}

func (d *LengthFieldFrameDecoder) Input(data []byte) error {
	i, length := 0, len(data)

	readLength := func() (int, error) {
		//拷贝包长
		if d.Size < d.fieldLength {
			n := libbufio.MinInt(d.fieldLength-d.Size, length)
			copy(d.Data[d.Size:], data[i:i+n])
			i += n
			d.Size += n

			if d.Size < d.fieldLength {
				return -1, nil
			}

			for i, v := range d.Data[:d.fieldLength] {
				d.total |= int(v) << ((len(d.Data[:d.fieldLength]) - i - 1) * 8)
			}

			if d.total == 0 {
				return -1, fmt.Errorf("the packet length cannot be 0")
			}

			d.total += d.fieldLength
			return d.total, nil
		}

		return 0, nil
	}

	for length-i > 0 {
		if d.Size < d.fieldLength {
			i2, err := readLength()

			if err != nil {
				return err
			}

			if i2 < 0 {
				return nil
			}
		}

		remain := length - i
		if remain < 1 {
			return nil
		}

		need := d.total - d.Size
		consume := libbufio.MinInt(remain, need)
		copy(d.Data[d.Size:], data[i:i+consume])
		i += consume
		d.Size += consume

		if remain < need {
			return nil
		}

		d.cb(d.Data[:d.Size])
		d.Size = 0
		d.total = 0
	}

	return nil
}

// DelimiterFrameDecoder 分隔符解码器
type DelimiterFrameDecoder struct {
	delimiter       []byte //分隔符
	delimiterLength int    //分隔符长度
	foundCount      int    //已经匹配到的分割符数量

	maxFrameLength int //最大帧长度

	cb   func([]byte)
	data []byte //解析缓存区
	size int    //已缓存数据长度
}

// NewDelimiterFrameDecoder 创建分隔符解码器
// @maxFrameLength 最大帧长, 如果在maxFrameLength范围内没解析完, 解析失败
func NewDelimiterFrameDecoder(maxFrameLength int, delimiter []byte, cb func([]byte)) *DelimiterFrameDecoder {
	utils.Assert(maxFrameLength > len(delimiter))
	return &DelimiterFrameDecoder{
		delimiter:       delimiter,
		delimiterLength: len(delimiter),
		maxFrameLength:  maxFrameLength,
		cb:              cb,
	}
}

func (d *DelimiterFrameDecoder) Input(data []byte) error {
	var offset int
	for i, v := range data {
		if d.delimiter[d.foundCount] != v {
			d.foundCount = 0
			continue
		}

		d.foundCount++
		if d.foundCount < d.delimiterLength {
			continue
		}

		//回调数据
		n := i + 1 - d.delimiterLength
		if d.size > 0 {
			//拷贝并回调
			if n > 0 {
				if d.maxFrameLength < d.size+n {
					return fmt.Errorf("frame length exceeds %d", d.maxFrameLength)
				}

				copy(d.data[d.size:], data[offset:n])
				d.size += n
			} else {
				//说明缓存包末尾包含分割符
				d.size -= n
			}

			d.cb(d.data[:d.size])
			d.size = 0
		} else if n > 0 {
			//回调当前包的数据
			d.cb(data[offset:n])
		}

		offset = i + 1
		d.foundCount = 0
	}

	//有未解析完的数据, 将剩余数据缓存起来
	n := len(data) - offset
	if n > 0 {
		if d.maxFrameLength < d.size+n {
			return fmt.Errorf("frame length exceeds %d", d.maxFrameLength)
		}
		if d.data == nil {
			d.data = make([]byte, d.maxFrameLength)
		}

		copy(d.data[d.size:], data[offset:])
		d.size += n
	}

	return nil
}
