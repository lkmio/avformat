package avformat

import "github.com/lkmio/avformat/utils"

type DataPipeline interface {
	Write(data []byte, index int, mediaType utils.AVMediaType) (int, error)

	Fetch(index int) ([]byte, error)

	DiscardBackPacket(index int)

	DiscardHeadPacket(index int)

	PendingBlockSize(index int) int
}
