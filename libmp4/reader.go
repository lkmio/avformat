package libmp4

import (
	"encoding/binary"
	"github.com/lkmio/avformat/libbufio"
)

type reader struct {
	data        []byte
	offset      int64
	isLargeSize bool
}

func newReader(data []byte) *reader {
	return &reader{data: data, offset: 0}
}

// func (r reader) hasNext() bool {
func (r *reader) nextSize() int64 {
	remain := int64(len(r.data)) - r.offset
	if remain < 4 {
		return -1
	}

	size := int64(libbufio.BytesToUInt32(r.data[r.offset], r.data[r.offset+1], r.data[r.offset+2], r.data[r.offset+3]))
	r.isLargeSize = size == 1
	if size == 0 {
		return 0
	} else if size == 1 {
		if remain < 8 {
			return -1
		}

		size = int64(binary.BigEndian.Uint64(r.data[r.offset+4:]))
	}

	if size <= remain {
		return size
	} else {
		return -1
	}
}

func (r *reader) next(size int64) (string, int64) {
	var temp int64
	if r.isLargeSize {
		temp = r.offset + 16
		size -= 12
	} else {
		temp = r.offset + 8
		size -= 8
	}

	r.offset = temp + size
	return string(r.data[temp-4 : temp]), size
}
