package avformat

import (
	"github.com/lkmio/avformat/collections"
	"github.com/lkmio/avformat/utils"
)

type StreamsBuffer struct {
	buffers []*collections.RBBlockBuffer
}

func (s *StreamsBuffer) findOrCreateStreamBuffer(index int, mediaType utils.AVMediaType) *collections.RBBlockBuffer {
	exist := index < len(s.buffers)
	if !exist {
		if utils.AVMediaTypeVideo == mediaType {
			s.buffers = append(s.buffers, collections.NewRBBlockBuffer(1024*1024*2))
		} else {
			s.buffers = append(s.buffers, collections.NewRBBlockBuffer(48000*12))
		}
	}

	return s.buffers[index]
}

func (s *StreamsBuffer) Write(data []byte, index int, mediaType utils.AVMediaType) (int, error) {
	s.findOrCreateStreamBuffer(index, mediaType).Write(data)
	return len(data), nil
}

func (s *StreamsBuffer) Feat(index int) ([]byte, error) {
	data := s.findOrCreateStreamBuffer(index, utils.AVMediaTypeUnknown).Feat()
	return data, nil
}

func (s *StreamsBuffer) Discard(index int) {
	s.buffers[index].Pop()
}

func (s *StreamsBuffer) DiscardBackPacket(index int) {
	s.buffers[index].PopBack()
}

func (s *StreamsBuffer) DiscardHeadPacket(index int) {
	s.buffers[index].Pop()
}

func (s *StreamsBuffer) PendingBlockSize(index int) int {
	return s.findOrCreateStreamBuffer(index, utils.AVMediaTypeUnknown).PendingBlockSize()
}
