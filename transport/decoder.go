package transport

import (
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
)

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
