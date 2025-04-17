package avformat

type Track interface {
	GetStream() *AVStream
}

type SimpleTrack struct {
	Stream *AVStream
}

func (s SimpleTrack) GetStream() *AVStream {
	return s.Stream
}
