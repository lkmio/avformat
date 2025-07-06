package avformat

import (
	"fmt"
	"github.com/lkmio/avformat/utils"
	"os"
	"strconv"
)

type OnUnpackStreamHandler interface {
	// OnNewTrack 新的track回调
	OnNewTrack(track Track)

	// OnTrackComplete track解析完毕回调
	OnTrackComplete()

	// OnTrackNotFind 未找到track回调
	OnTrackNotFind()

	OnPacket(packet *AVPacket)
}

type OnUnpackStreamLogger struct {
	tracks TrackManager
}

func (o *OnUnpackStreamLogger) OnNewTrack(track Track) {
	utils.Assert(o.tracks.Add(track))

	if utils.AVMediaTypeAudio == track.GetStream().MediaType {
		fmt.Printf("tack type: %s codec: %s index: %d sample_rate: %d channels: %d\r\n", track.GetStream().MediaType.String(), track.GetStream().CodecID.String(), track.GetStream().Index, track.GetStream().SampleRate, track.GetStream().Channels)
	} else {
		fmt.Printf("tack type: %s codec: %s index: %d\r\n", track.GetStream().MediaType.String(), track.GetStream().CodecID.String(), track.GetStream().Index)
	}
}

func (o *OnUnpackStreamLogger) OnTrackComplete() {
	fmt.Printf("track complete\r\n")
}

func (o *OnUnpackStreamLogger) OnPacket(packet *AVPacket) {
	fmt.Printf("packet type: %s dts: %d pts: %d\r\n", packet.MediaType.String(), packet.Dts, packet.Pts)
}

func (o *OnUnpackStreamLogger) OnTrackNotFind() {
	println("track not find")
}

type OnUnpackStream2FileHandler struct {
	OnUnpackStreamLogger
	Path string
	fos  []*os.File
}

func (o *OnUnpackStream2FileHandler) OnNewTrack(track Track) {
	o.OnUnpackStreamLogger.OnNewTrack(track)
	stream := track.GetStream()
	file, err := os.OpenFile(o.Path+"."+stream.MediaType.String()+"."+strconv.Itoa(stream.Index)+"."+stream.CodecID.String(), os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}

	o.fos = append(o.fos, file)
}

func (o *OnUnpackStream2FileHandler) OnTrackComplete() {
	o.OnUnpackStreamLogger.OnTrackComplete()
}

func (o *OnUnpackStream2FileHandler) OnPacket(packet *AVPacket) {
	o.OnUnpackStreamLogger.OnPacket(packet)

	data := packet.Data
	if utils.AVMediaTypeVideo == packet.MediaType {
		stream := o.tracks.Get(packet.Index).GetStream()
		data = AVCCPacket2AnnexB(stream, packet)
		if packet.Key && PacketTypeAVCC == packet.PacketType && stream.CodecParameters != nil {
			extraData := stream.CodecParameters.AnnexBExtraData()
			if _, err := o.fos[packet.Index].Write(extraData); err != nil {
				panic(err)
			}
		}
	}

	if _, err := o.fos[packet.Index].Write(data); err != nil {
		panic(err)
	}
}
