package librtsp

import (
	"github.com/lkmio/avformat/utils"
	"os"
	"testing"
)

func TestPuller(t *testing.T) {
	h264File, _ := os.OpenFile("../rtsp.h264", os.O_WRONLY|os.O_CREATE, 132)
	defer func() {
		h264File.Close()
	}()

	puller := NewPuller(func(mediaType utils.AVMediaType, data []byte) {
		if mediaType == utils.AVMediaTypeVideo {
			h264File.Write(data[12:])
		}
	})
	//url := "rtsp://wowzaec2demo.streamlock.net:554/vod/mp4:BigBuckBunny_115k.mov"
	url := "rtsp://192.168.2.148/hls/mystream"
	puller.Open(url)
	select {}
}
