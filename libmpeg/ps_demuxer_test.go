package libmpeg

import (
	"bufio"
	"github.com/yangjiechina/avformat/utils"
	"os"
	"testing"
)

func TestDecodePS(t *testing.T) {
	args := os.Args
	path := args[len(args)-1]

	fileObj, err := os.OpenFile(path+".video", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		fileObj.Close()
	}()

	//count := 0
	deMuxer := NewPSDeMuxer()

	deMuxer.SetHandler(func(data []byte, total int, first bool, mediaType utils.AVMediaType, id utils.AVCodecID, dts int64, pts int64, params interface{}) error {
		if utils.AVMediaTypeVideo == mediaType {
			fileObj.Write(data)
		}
		return nil
	})

	open, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	bytes := make([]byte, 1500)
	reader := bufio.NewReader(open)
	var offset int
	for n, err := reader.Read(bytes[offset:]); n > 0 && err == nil; n, err = reader.Read(bytes[offset:]) {
		end := offset + n
		consume, err := deMuxer.Input(bytes[:end])
		if err != nil {
			panic(err)
		}

		offset = end - consume
		utils.Assert(offset < len(bytes))

		if offset > 0 {
			copy(bytes, bytes[end-offset:end])
		}
	}
}
