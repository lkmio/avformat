package libmp4

import (
	"github.com/yangjiechina/avformat/libhevc"
	"github.com/yangjiechina/avformat/utils"
	"os"
	"testing"
)

func TestMp4DeMuxer(t *testing.T) {
	//path := "../232937384-1-208_baseline.mp4"
	path := "../LB1l2iXISzqK1RjSZFjXXblCFXa.mp4"
	h264File, err := os.OpenFile(path+".h264", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		h264File.Close()
	}()

	h265File, err := os.OpenFile(path+".h265", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		h265File.Close()
	}()

	aacFile, err := os.OpenFile(path+".aac", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		aacFile.Close()
	}()

	convertBuffer := utils.NewByteBuffer()
	var videoTrack *Track
	var audioTrack *Track
	var config *utils.MPEG4AudioConfig
	header := make([]byte, 7)

	muxer := NewDeMuxer(func(data []byte, pts, dts int64, mediaType utils.AVMediaType, id utils.AVCodecID) {
		switch id {
		case utils.AVCodecIdH264:
			utils.Mp4ToAnnexB(convertBuffer, data, videoTrack.MetaData().ExtraData())
			convertBuffer.ReadTo(func(bytes []byte) {
				h264File.Write(bytes)
			})
			break
		case utils.AVCodecIdH265:
			libhevc.Mp4ToAnnexB(convertBuffer, data, videoTrack.MetaData().ExtraData(), videoTrack.MetaData().(*VideoMetaData).LengthSize())
			convertBuffer.ReadTo(func(bytes []byte) {
				h265File.Write(bytes)
			})
			break
		case utils.AVCodecIdAAC:
			utils.SetADtsHeader(header, 0, config.ObjectType-1, config.SamplingIndex, config.ChanConfig, 7+(len(data)))
			aacFile.Write(header)
			aacFile.Write(data)
			break
		}

		convertBuffer.Clear()
	})

	if err := muxer.Open(path); err != nil {
		panic(err)
	}

	if tracks := muxer.FindTrack(utils.AVMediaTypeVideo); tracks == nil {
		panic("Not find for video track.")
	} else {
		videoTrack = tracks[0]
	}

	if tracks := muxer.FindTrack(utils.AVMediaTypeAudio); tracks != nil {
		audioTrack = tracks[0]
		metaData := audioTrack.MetaData()
		if audioTrack != nil && metaData.CodeId() == utils.AVCodecIdAAC {
			config, err = utils.ParseMpeg4AudioConfig(metaData.ExtraData())
			if err != nil {
				panic(err)
			}
		}
	}

	for err = muxer.Read(); err == nil; err = muxer.Read() {

	}
	//muxer.Read("../LB1l2iXISzqK1RjSZFjXXblCFXa.mp4")
}
