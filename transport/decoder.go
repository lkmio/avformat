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
	data []byte //本次输入的数据长度小于帧长时, 作为缓存区
	size int    //缓存区大小

	cb func([]byte)

	maxFrameLength   int //最大帧长
	frameLength      int //当前帧长
	fieldLength      int //几个字节描述数据帧长
	fieldLengthCount int //已经读取到几个字节帧长
}

// NewLengthFieldFrameDecoder 创建帧长解码器
// @maxFrameLength 最大帧长, 如果在maxFrameLength范围内没解析完, 解析失败
// @fieldLength 几个字节描述帧长
func NewLengthFieldFrameDecoder(maxFrameLength, fieldLength int, cb func([]byte)) *LengthFieldFrameDecoder {
	utils.Assert(maxFrameLength > 0)
	utils.Assert(fieldLength > 0 && fieldLength < 5)

	return &LengthFieldFrameDecoder{maxFrameLength: maxFrameLength, cb: cb, fieldLength: fieldLength}
}

func (d *LengthFieldFrameDecoder) callback(data []byte) {
	d.cb(data)

	//清空标记,重新读取
	d.frameLength = 0
	d.fieldLengthCount = 0
}

func (d *LengthFieldFrameDecoder) Input(data []byte) error {
	var index int
	length := len(data)

	for index < length {
		//读取帧长度
		for ; d.fieldLengthCount < d.fieldLength && index < length; index++ {
			d.frameLength = d.frameLength<<8 | int(data[index])
			d.fieldLengthCount++
		}

		//不够帧长
		if d.fieldLengthCount < d.fieldLength {
			return nil
		}

		if d.frameLength > d.maxFrameLength {
			return fmt.Errorf("frame length exceeds %d", d.maxFrameLength)
		}

		n := length - index

		//有缓存数据或者数据不够缓存起来
		if d.size > 0 || n < d.frameLength {
			if d.data == nil {
				d.data = make([]byte, d.maxFrameLength)
			}

			consume := libbufio.MinInt(d.frameLength-d.size, n)
			copy(d.data[d.size:], data[index:index+consume])
			d.size += consume
			index += consume
		}

		if d.size >= d.frameLength {
			//回调缓存数据
			d.callback(d.data[:d.frameLength])
			d.size = 0
		} else if n >= d.frameLength {
			//免拷贝回调
			index += d.frameLength
			d.callback(data[index-d.frameLength : index])
		}
	}

	return nil
}

// DelimiterFrameDecoder 分隔符解码器
type DelimiterFrameDecoder struct {
	delimiter       []byte //分隔符
	delimiterLength int    //分隔符长度

	foundCount     int //已经匹配到的分割符数量
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
		//匹配分隔符
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
		//回调缓存数据
		if d.size > 0 {
			//拷贝本次读取的数据
			if n > 0 {
				if d.maxFrameLength < d.size+n {
					return fmt.Errorf("frame length exceeds %d", d.maxFrameLength)
				}

				copy(d.data[d.size:], data[offset:n])
				d.size += n
			} else {
				//缓存包末尾包含分割符
				d.size -= n
			}

			d.cb(d.data[:d.size])
			d.size = 0
		} else if n > 0 {
			//免拷贝回调当前包数据
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
