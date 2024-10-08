package libhevc

import (
	"fmt"
	"github.com/lkmio/avformat/libavc"
	"os"
	"testing"
)

func TestUtil(t *testing.T) {
	file, err := os.ReadFile("../h265.hevc")
	if err != nil {
		panic(err)
	}

	var index int
	var lastKeyFrameIndex = -1
	libavc.SplitNalU(file, func(nalu []byte) {
		index++
		if IsKeyFrame(nalu) {
			if lastKeyFrameIndex != -1 {
				println(fmt.Sprintf("关键帧间隔:%d", index-lastKeyFrameIndex))
			}
			lastKeyFrameIndex = index
		}
	})
}
