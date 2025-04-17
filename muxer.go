package avformat

import "fmt"

type Muxer interface {
	AddTrack(stream *AVStream) (int, error)

	WriteHeader(dst []byte) (int, error)

	Input(dst []byte, index int, data []byte, dts, pts int64) (int, error)
}

type BaseMuxer struct {
	Tracks    TrackManager
	Completed bool
}

func (b *BaseMuxer) AddTrack(track Track) (int, error) {
	old := b.Tracks.FindTrackWithType(track.GetStream().MediaType)
	if old != nil {
		return -1, fmt.Errorf("track with media type '%s' already exists", track.GetStream().MediaType)
	}

	_ = b.Tracks.Add(track)

	return b.Tracks.Size() - 1, nil
}

func (b *BaseMuxer) WriteHeader(_ []byte) (int, error) {
	b.Completed = true
	return 0, nil
}

func (b *BaseMuxer) Input(dst []byte, index int, data []byte, dts, pts int64) (int, error) {
	//TODO implement me
	panic("implement me")
}
