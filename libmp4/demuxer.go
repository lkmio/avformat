package libmp4

import (
	"fmt"
	"github.com/lkmio/avformat/utils"
	"io"
	"io/ioutil"
)

type deMuxHandler func(data []byte, pts, dts int64, mediaType utils.AVMediaType, id utils.AVCodecID)

type DeMuxer struct {
	ctx          *deMuxContext
	reader       *utils.FileReader
	sampleBuffer []byte
	handler      deMuxHandler
}

func NewDeMuxer(handler deMuxHandler) *DeMuxer {
	return &DeMuxer{handler: handler}
}

type deMuxContext struct {
	root   *file
	tracks []*Track
}

func (d *DeMuxer) recursive(ctx *deMuxContext, parent box, data []byte) (bool, error) {
	r := newReader(data)
	var size int64
	for size = r.nextSize(); size > 0; size = r.nextSize() {
		name, n := r.next(size)
		fmt.Printf("size:%d name:%s\r\n", size, name)

		if parse, ok := parsers[name]; !ok {
			return false, fmt.Errorf("unknow box type:%s", name)
		} else {
			b, consume, err := parse(ctx, data[r.offset-n:r.offset])
			if err != nil {
				return false, err
			}

			parent.addChild(b)
			if b.hasContainer() {
				_, e := d.recursive(ctx, b, data[r.offset-n+int64(consume):r.offset])
				if e != nil {
					return false, e
				}
			}
		}
	}

	//Not the last box. need more...
	if size != 0 {
		return false, nil
	}

	return true, nil
}

func (d *DeMuxer) findNextTrack() *Track {
	var trak *Track
	for _, t := range d.ctx.tracks {
		if t.currentSample >= t.sampleCount {
			continue
		}

		if trak == nil || trak != nil && t.sampleIndexEntries[t.currentSample].pos < trak.sampleIndexEntries[trak.currentSample].pos {
			trak = t
		}
	}

	return trak
}

func buildIndex(ctx *deMuxContext) error {
	length := len(ctx.tracks)
	if length == 0 {
		return fmt.Errorf("uninvalid data")
	}

	for _, t := range ctx.tracks {
		if t.mark>>26 != 0x3F {
			return fmt.Errorf("uninvalid data")
		}

		t.sampleCount = t.stsz.sampleCount
		t.chunkCount = t.stco.entryCount
		t.sampleIndexEntries = make([]*sampleIndexEntry, t.sampleCount)
		var index uint32
		var duration uint32
		var dts int64
		addSampleIndex := func(chunkOffsetIndex, size int) {
			chunkOffset := t.stco.chunkOffset[chunkOffsetIndex]
			var sampleOffset uint32
			for n := 0; n < size; n++ {
				entry := sampleIndexEntry{}
				entry.pos = int64(chunkOffset + sampleOffset)
				entry.size = t.stsz.entrySize[index]
				if t.stss != nil {
					_, ok := t.stss.sampleNumber[index+1]
					entry.keyFrame = ok
				}

				tempIndex := index
				for i := 0; i < len(t.stts.sampleCount); i++ {
					if tempIndex < t.stts.sampleCount[i] {
						duration = t.stts.sampleDelta[i]
						break
					} else {
						tempIndex -= t.stts.sampleCount[i]
					}
				}

				entry.timestamp = dts
				dts += int64(duration)

				sampleOffset += entry.size
				t.sampleIndexEntries[index] = &entry
				index++
			}
		}

		for i := 0; i < len(t.stsc.firstChunk); i++ {
			chunk := t.stsc.firstChunk[i]
			size := t.stsc.samplesPerChunk[i]
			// All subsequent chunks size
			if i+1 == len(t.stsc.firstChunk) {
				for ; chunk <= t.chunkCount; chunk++ {
					addSampleIndex(int(chunk-1), int(size))
				}
			} else {
				nextChunk := t.stsc.firstChunk[i+1]
				for ; chunk < nextChunk; chunk++ {
					addSampleIndex(int(chunk-1), int(size))
				}
			}

		}
	}

	return nil
}

func (d *DeMuxer) Read() error {
	nextTrack := d.findNextTrack()
	if nextTrack == nil {
		return io.EOF
	}

	entry := nextTrack.sampleIndexEntries[nextTrack.currentSample]

	if int(entry.size) > len(d.sampleBuffer) {
		d.sampleBuffer = make([]byte, entry.size)
	}

	if err := d.reader.Seek(entry.pos); err != nil {
		return err
	}

	if _, err := d.reader.Read(d.sampleBuffer[:entry.size]); err != nil {
		return err
	}
	d.handler(d.sampleBuffer[:entry.size], entry.timestamp, entry.timestamp, nextTrack.metaData.MediaType(), nextTrack.metaData.CodeId())
	nextTrack.currentSample++
	return nil
}

func (d *DeMuxer) TrackCount() int {
	return len(d.ctx.tracks)
}

func (d *DeMuxer) FindTrack(mediaType utils.AVMediaType) []*Track {
	var tracks []*Track
	for _, track := range d.ctx.tracks {
		if track.metaData.MediaType() == mediaType {
			tracks = append(tracks, track)
		}
	}

	return tracks
}

func (d *DeMuxer) Open(path string) error {
	all, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	context := &deMuxContext{}
	context.root = &file{}
	if _, err = d.recursive(context, context.root, all); err != nil {
		return err
	}

	d.ctx = context
	d.reader = &utils.FileReader{}
	_ = d.reader.Open(path)
	d.sampleBuffer = make([]byte, 1024*1024*2)
	return buildIndex(context)
}
