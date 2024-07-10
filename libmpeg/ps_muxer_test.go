package libmpeg

import (
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"os"
	"testing"
)

func TestPSMuxer(t *testing.T) {
	path := "../1.raw"
	fileObj, err := os.OpenFile(path+".ps", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		fileObj.Close()
	}()

	muxer := NewMuxer(func(index int, data []byte, pts, dts int64) {
		fileObj.Write(data)
	})
	streamIndex := make(map[int]int, 2)
	count := 0
	deMuxer := NewDeMuxer(func(buffer libbufio.ByteBuffer, keyFrame bool, streamType int, pts, dts int64) {
		fmt.Printf("count:%d type:%d length:%d keyFrame=%t pts:=%d dts:%d\r\n", count, streamType, buffer.Size(), keyFrame, pts, dts)
		count++
		index, ok := streamIndex[streamType]
		if !ok {
			i, err2 := muxer.AddStream(streamType)
			if err2 != nil {
				panic(err2)
			}
			streamIndex[streamType] = i
			index = i
		}
		muxer.Input(index, keyFrame, buffer.ToBytes(), pts, dts)
	})

	if err = deMuxer.Open(path, 0); err != nil {
		panic(err)
	} else {
		deMuxer.Close()
	}
}
