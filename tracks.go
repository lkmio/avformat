package avformat

import "github.com/lkmio/avformat/utils"

type TrackManager struct {
	Tracks []Track
}

func (t *TrackManager) Add(track Track) bool {
	//if t.FindTrackWithType(track.GetStream().MediaType) != nil {
	//	return false
	//}

	t.Tracks = append(t.Tracks, track)
	return true
}

func (t *TrackManager) Find(id utils.AVCodecID) Track {
	for _, track := range t.Tracks {
		if track.GetStream().CodecID == id {
			return track
		}
	}

	return nil
}

func (t *TrackManager) FindTrackWithType(mediaType utils.AVMediaType) Track {
	for _, track := range t.Tracks {
		if track.GetStream().MediaType == mediaType {
			return track
		}
	}

	return nil
}

func (t *TrackManager) Get(index int) Track {
	return t.Tracks[index]
}

func (t *TrackManager) Size() int {
	return len(t.Tracks)
}
