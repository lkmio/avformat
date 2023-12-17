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

type tsTrack struct {
	streamType int
	pes        *PESHeader
}
type tsMuxer struct {
	tracks []*tsTrack
}

func (t *tsMuxer) AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID) (int, error) {
	var pes *PESHeader
	if utils.AVMediaTypeAudio == mediaType {
		pes = NewPESPacket(StreamIdAudio)
	} else if utils.AVMediaTypeVideo == mediaType {
		pes = NewPESPacket(StreamIdVideo)
	} else {
		utils.Assert(false)
	}

	streamType, ok := codecId2StreamTypeMap[id]
	if !ok {
		return -1, fmt.Errorf("the codec %d does not support mux to ts stream", int(id))
	}

	for _, track := range t.tracks {
		utils.Assert(track.streamType != streamType)
	}

	t.tracks = append(t.tracks, &tsTrack{streamType: streamType, pes: pes})
	return len(t.tracks) - 1, nil
}

func (t *tsMuxer) WriteHeader() {
	utils.Assert(len(t.tracks) > 0)

	bytes := make([]byte, 1024)
	n := generatePAT(bytes, 0)
	utils.Assert(n > 0 && n < TsPacketSize)
	copy(bytes[n:], stuffing[n:])

	streamTypes := make([][2]int16, len(t.tracks))
	for index, track := range t.tracks {
		streamTypes[index][0] = int16(track)
		streamTypes[index][1] = int16(0x100 + index)
	}

	n = generatePMT(bytes[TsPacketSize:], 0, streamTypes)
	utils.Assert(n > 0 && n < TsPacketSize)
	copy(bytes[TsPacketSize+n:], stuffing[:TsPacketSize-n])
}

func (t *tsMuxer) Input(trackIndex int, data []byte, dts, pts int64) error {
	track := t.tracks[trackIndex]

	track.pes.packetLength = 0x0000
	track.pes.dts = dts
	track.pes.pts = pts

	bytes := make([]byte, TsPacketSize)
	n := track.pes.ToBytes(bytes)
	count := len(data) + n/TsPacketSize

	for i := 0; i < count; i++ {

	}

	return nil
}

func (t *tsMuxer) Reset() {

}
