package libmpeg

import (
	"fmt"
	"github.com/yangjiechina/avformat/libbufio"
	"math"
)

const muxerBufferSize = 1024 * 1024 * 2
const pesPacketSize = 65535 - 100

type encodeHandler func(index int, data []byte, pts, dts int64)

type stream struct {
	streamType int
	pesPacket  PESHeader
}

func (s stream) isAudio() bool {
	return s.pesPacket.streamId == StreamIdAudio
}

type Muxer struct {
	buffer       []byte
	handler      encodeHandler
	streams      []stream
	packetHeader PacketHeader
	systemHeader SystemHeader
	psm          ProgramStreamMap
}

func NewMuxer(handler encodeHandler) *Muxer {
	m := &Muxer{
		handler: handler,
		buffer:  make([]byte, muxerBufferSize),
	}
	m.packetHeader.programMuxRate = 6106
	m.systemHeader.rateBound = 26234
	return m
}

func (r *Muxer) AddStream(streamType int) (int, error) {
	streamId, ok := streamTypes[streamType]
	if !ok {
		return -1, fmt.Errorf("unknow stream type %d", streamType)
	}

	for _, s := range r.streams {
		if s.streamType == streamType {
			return -1, fmt.Errorf("must be unique for stream type")
		}
	}

	s := StreamType(streamType)
	if s.isAudio() {
		r.systemHeader.streams = append(r.systemHeader.streams,
			streamHeader{
				streamId:         StreamIdAudio,
				bufferBoundScale: 0,
				bufferSizeBound:  32,
			},
		)

		r.psm.elementaryStreams = append(r.psm.elementaryStreams,
			ElementaryStream{
				streamType: byte(streamType),
				streamId:   StreamIdAudio,
				info:       nil,
			})

		r.systemHeader.audioBound++
	} else if s.isVideo() {
		r.systemHeader.streams = append(r.systemHeader.streams,
			streamHeader{
				streamId:         StreamIdVideo,
				bufferBoundScale: 1,
				bufferSizeBound:  400,
			},
		)

		r.psm.elementaryStreams = append(r.psm.elementaryStreams,
			ElementaryStream{
				streamType: byte(streamType),
				streamId:   StreamIdVideo,
				info:       nil,
			})

		r.systemHeader.videoBound++
	} else {
		panic("")
	}

	r.streams = append(r.streams, stream{streamType: streamType})
	r.streams[len(r.streams)-1].pesPacket.streamId = byte(streamId)
	r.streams[len(r.streams)-1].pesPacket.dataAlignmentIndicator = 1
	return len(r.streams) - 1, nil
}

// Input
// @pts must be >= 0
// @dts if the DTS does not exist. DTS must be < 0
func (r *Muxer) Input(index int, keyFrame bool, data []byte, pts, dts int64) {
	if pts < 0 {
		panic("PTS must be than 0")
	}

	var n int
	var i int
	s := r.streams[index]
	s.pesPacket.pts = pts
	s.pesPacket.dts = dts

	//add pack header
	if dts >= 3600 {
		r.packetHeader.systemClockReferenceBase = dts - 3600
	} else {
		r.packetHeader.systemClockReferenceBase = 0
	}

	n = r.packetHeader.ToBytes(r.buffer)
	i += n
	//add system header and psm
	if keyFrame {
		n = r.systemHeader.ToBytes(r.buffer[i:])
		i += n
		n = r.psm.ToBytes(r.buffer[i:])
		i += n
	}

	s.pesPacket.ptsDtsFlags = 0x2
	if dts >= 0 {
		s.pesPacket.ptsDtsFlags = 0x3
	}

	length := len(data)
	pesCount := int(math.Ceil(float64(length) / pesPacketSize))
	for j := 0; j < pesCount; j++ {
		size := libbufio.MinInt(length, pesPacketSize)
		n = s.pesPacket.ToBytes(r.buffer[i:])

		libbufio.WriteWORD(r.buffer[i+4:], uint16(size+n-6))
		i += n
		copy(r.buffer[i:], data[j*pesPacketSize:j*pesPacketSize+size])
		i += size
		length -= size

		//Only the first packet contains PTS and DTS.
		s.pesPacket.ptsDtsFlags = 0x00
	}

	r.handler(index, r.buffer[:i], pts, dts)
}

// InputWithMix 视频和音频打包到同一个ps包
func (r *Muxer) InputWithMix(keyFrame bool, videoData, audioData []byte, pts, dts int64) {

}

func (r *Muxer) Close() {

}
