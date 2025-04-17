package avformat

import (
	"github.com/lkmio/avformat/bufio"
	"github.com/lkmio/avformat/utils"
)

type DirectDataPipeline struct {
	data []byte
	size int
}

func (d *DirectDataPipeline) Write(data []byte, index int, mediaType utils.AVMediaType) (int, error) {
	n := cap(d.data) - d.size
	length := len(data)
	if n < length {
		bytes := make([]byte, (d.size+length)*2)
		copy(bytes, d.data[:d.size])
		d.data = bytes
	}

	copy(d.data[d.size:], data)
	d.size += length

	return d.size, nil
}

func (d *DirectDataPipeline) Feat(index int) ([]byte, error) {
	bytes := d.data[:d.size]
	d.size = 0
	return bytes, nil
}

func (d *DirectDataPipeline) Seek(offset int64, index int) error {
	d.size = bufio.MaxInt(d.size+int(offset), 0)
	return nil
}

func (d *DirectDataPipeline) PendingBlockSize(index int) int {
	return d.size
}

func (d *DirectDataPipeline) DiscardBackPacket(index int) {
}

func (d *DirectDataPipeline) DiscardHeadPacket(index int) {
}
