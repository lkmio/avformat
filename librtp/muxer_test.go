package librtp

import (
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/libmpeg"
	"net"
	"testing"
)

func TestRTPMuxer(t *testing.T) {
	path := "../1.raw"

	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.11.100"),
		Port: 50000,
	}
	socket, err := net.DialUDP("udp4", nil, remoteAddr)

	rtpMuxer := NewMuxer(96, 0, 0xFFFFFFFF, func(data []byte, timestamp uint32) {
		_, err2 := socket.Write(data)
		if err2 != nil {
			panic(err2)
		}
	})

	muxer := libmpeg.NewMuxer(func(index int, data []byte, pts, dts int64) {
		rtpMuxer.Input(data, uint32(pts))
	})

	streamIndex := make(map[int]int, 2)
	count := 0
	deMuxer := libmpeg.NewDeMuxer(func(buffer libbufio.ByteBuffer, keyFrame bool, streamType int, pts, dts int64) {
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
