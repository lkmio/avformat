package libmpeg

/*import (
	"github.com/lkmio/avformat/utils"
	"io/ioutil"
	"os"
	"testing"
)

func TestTSDeMuxer(t *testing.T) {
	path := "../sample_1280x720_surfing_with_audio.ts"
	file, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	videoOutputFile, _ := os.OpenFile(path+".video", os.O_WRONLY|os.O_CREATE, 132)
	defer func() {
		videoOutputFile.Close()
	}()

	audioOutputFile, _ := os.OpenFile(path+".audio", os.O_WRONLY|os.O_CREATE, 132)
	defer func() {
		audioOutputFile.Close()
	}()

	muxer := NewTSDeMuxer(func(buffer utils.ByteBuffer, keyFrame bool, streamType int, pts, dts int64) {
		if streamType == StreamIdH624 || streamType == StreamIdVideo {
			buffer.ReadTo(func(bytes []byte) {
				videoOutputFile.Write(bytes)
			})
		} else if streamType == StreamIdAudio {
			buffer.ReadTo(func(bytes []byte) {
				audioOutputFile.Write(bytes)
			})
		}
	})

	length := len(file)
	for count := 0; length >= 188; count++ {
		i := len(file) - length
		err := muxer.doRead(file[i : i+188])
		if err != nil {
			panic(err)
		}
		length -= 188
	}

}*/
