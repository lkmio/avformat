package libmpeg

import (
	"github.com/yangjiechina/avformat/utils"
	"io/ioutil"
	"os"
	"testing"
)

func TestTSMuxer(t *testing.T) {
	//输入H264裸流，循环读取写入TS文件.
	args := os.Args
	path := args[len(args)-1]
	file, err2 := ioutil.ReadFile(path)
	utils.Assert(err2 == nil)

	fileObj, err := os.OpenFile(path+".ts", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		fileObj.Close()
	}()

	muxer := NewTSMuxer()
	writeBuffer := make([]byte, 1024*1024)
	var bufferSize = 0
	muxer.SetAllocHandler(func(size int) []byte {
		n := len(writeBuffer) - bufferSize
		if n < 0 {
			utils.Assert(false)
		}
		if n < size {
			fileObj.Write(writeBuffer[:bufferSize])
			bufferSize = 0
		}

		return writeBuffer[bufferSize : bufferSize+size]
	})

	muxer.SetWriteHandler(func(data []byte) {
		bufferSize += len(data)
	})

	videoTrackIndex, err := muxer.AddTrack(utils.AVMediaTypeVideo, utils.AVCodecIdH264)
	utils.Assert(err == nil)

	muxer.WriteHeader()

	var start = 0
	var pts = int64(3600)
	var mark = 0
	for i, b := range file {
		if b == 0 {
			mark++
			continue
		} else if b == 1 && mark > 1 {
			mark++
			continue
		} else if b == 0x41 && mark > 2 {

		} else {
			mark = 0
			continue
		}

		pkt := file[start : i-mark]
		start = i - mark

		muxer.Input(videoTrackIndex, pkt, pts, pts)
		pts += 3600
		mark = 0
	}

	if bufferSize > 0 {
		fileObj.Write(writeBuffer[:bufferSize])
		bufferSize = 0
	}
}
