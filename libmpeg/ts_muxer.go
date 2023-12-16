package libmpeg

import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
)

type TSMuxer interface {
	AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID) (int, error)

	WriteHeader()

	Input(trackIndex int, data []byte, dts, pts int64) error

	Reset()
}

func NewTSMuxer() TSMuxer {
	return &tsMuxer{}
}

type tsMuxer struct {
	tracks []int
}

func (t *tsMuxer) AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID) (int, error) {
	if utils.AVMediaTypeAudio == mediaType {

	} else if utils.AVMediaTypeVideo == mediaType {

	} else {
		utils.Assert(false)
	}

	streamType, ok := codecId2StreamTypeMap[id]
	if !ok {
		return -1, fmt.Errorf("the codec %d does not support mux to ts stream", int(id))
	}

	for _, track := range t.tracks {
		utils.Assert(track != streamType)
	}

	t.tracks = append(t.tracks, streamType)
	return len(t.tracks) - 1, nil
}

func (t *tsMuxer) WriteHeader() {
	generatePAT()
	generatePMT()

}

func (t *tsMuxer) Input(trackIndex int, data []byte, dts, pts int64) error {

	return nil
}

func (t *tsMuxer) Reset() {

}
