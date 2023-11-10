package libflv

import (
	"bufio"
	"fmt"
	"github.com/yangjiechina/avformat/utils"
	"os"
	"testing"
)

type Handler struct {
	streams []utils.AVStream
	first   bool
	muxer   *Muxer
	out     *os.File
}

func (h *Handler) OnDeMuxStream(stream utils.AVStream) {
	h.streams = append(h.streams[:], stream)
}

func (h *Handler) OnDeMuxStreamDone() {

}

func (h *Handler) OnDeMuxPacket(index int, packet *utils.AVPacket2) {
	println(fmt.Sprintf("OnDeMuxPacket dts:%d pts:%d", packet.Dts(), packet.Pts()))

	if h.first {
		h.first = false
		var audioStream utils.AVStream
		var videoStream utils.AVStream
		var audioCodecId utils.AVCodecID
		var videoCodecId utils.AVCodecID
		for _, stream := range h.streams {
			if utils.AVMediaTypeAudio == stream.Type() {
				audioStream = stream
				audioCodecId = audioStream.CodecId()
			} else if utils.AVMediaTypeVideo == stream.Type() {
				videoStream = stream
				videoCodecId = videoStream.CodecId()
			}
		}

		h.muxer = NewMuxer(audioCodecId, videoCodecId, 0, 0, 0)
		var header_ [512]byte
		n := h.muxer.WriteHeader(header_[:])

		if audioStream != nil {
			n += h.muxer.Input(header_[n:], utils.AVMediaTypeAudio, len(audioStream.Extra()), 0, 0, false, true)
			copy(header_[n:], audioStream.Extra())
			n += len(audioStream.Extra())
		}

		if videoStream != nil {
			n += h.muxer.Input(header_[n:], utils.AVMediaTypeVideo, len(videoStream.Extra()), 0, 0, false, true)
			copy(header_[n:], videoStream.Extra())
			n += len(videoStream.Extra())
		}

		h.out.Write(header_[:])
	}

	var tagHeader [64]byte
	n := h.muxer.Input(tagHeader[:], packet.MediaType(), len(packet.Data()), packet.Dts(), packet.Pts(), packet.KeyFrame(), false)
	bytes := append(tagHeader[:n], packet.Data()...)

	h.out.Write(bytes)
}

func (h *Handler) OnDeMuxDone() {
}

func TestDeMuxer(t *testing.T) {
	args := os.Args
	path := args[len(args)-1]

	h264File, err := os.OpenFile(path+".h264", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		h264File.Close()
	}()

	aacFile, err := os.OpenFile(path+".aac", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}

	defer func() {
		aacFile.Close()
	}()

	outfile, err := os.OpenFile(path+".muxer.flv", os.O_WRONLY|os.O_CREATE, 132)

	if err != nil {
		panic(err)
	}

	defer func() {
		outfile.Close()
	}()

	muxer := DeMuxer{}
	handler := &Handler{first: true, out: outfile}
	muxer.SetHandler(handler)

	open, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	bytes := make([]byte, 1024*1024*2)
	reader := bufio.NewReader(open)
	var size int
	var n int

	for n, err = reader.Read(bytes[size:]); err == nil && n > 0; n, err = reader.Read(bytes[size:]) {
		size += n
		if consume, err := muxer.Input(bytes[:size]); err != nil {
			panic(err)
		} else {
			size -= consume
		}
	}
}
