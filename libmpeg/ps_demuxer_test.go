package libmpeg

/*import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
	"os"
	"testing"
)

func TestDecodePS(t *testing.T) {
	//path := "../1.raw"
	path := "D:/CProjects/test3/test7/yushi.raw.video"
	fileObj, err := os.OpenFile(path+".h264", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		fileObj.Close()
	}()
	count := 0
	deMuxer := NewDeMuxer(func(buffer utils.ByteBuffer, keyFrame bool, streamType int, pts, dts int64) {
		fmt.Printf("count:%d  type:%d length:%d keyFrame=%t pts:=%d dts:%d\r\n", count, streamType, buffer.Size(), keyFrame, pts, dts)
		count++
		buffer.ReadTo(func(bytes []byte) {
			fileObj.Write(bytes)
		})
	})

	if err = deMuxer.Open(path, 0); err != nil {
		//if err = deMuxer.Open(path, 1024*1024*2); err != nil {

		panic(err)
	} else {
		deMuxer.Close()
	}
}
*/
