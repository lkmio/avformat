package librtmp

import (
	"os"
	"testing"
)

func TestRTMPPuller(t *testing.T) {
	//url := "rtmp://ns8.indexforce.com/home/mystream"
	url := "rtmp://192.168.2.139/hls/mystream"
	//url := "rtmp://192.168.2.146/hls/test"
	h264File, err := os.OpenFile("../rtmp.h264", os.O_WRONLY|os.O_CREATE, 132)
	if err != nil {
		panic(err)
	}
	defer func() {
		h264File.Close()
	}()

	/*	videoCallbackBuffer := make([]byte, 1024*1204)
		videoBufferLength := 0
		videoLastPts := 0
		var extraData []byte*/
	//audioConfig := &utils.MPEG4AudioConfig{}
	//aacADtsHeader := make([]byte, 7)
	var videoTS int
	var audioTS int
	puller := NewPuller(func(data []byte, ts int) {
		videoTS += ts
		//println(fmt.Sprintf("video data lenth:%d ts:%d", len(data), videoTS))
		/*		if ts != 0 {
					//payload
					codecId := libflv.VideoCodecId(videoCallbackBuffer[0] & 0xF)
					if codecId == libflv.VideoCodeIdH264 {
						pktType := videoCallbackBuffer[1]
						ct := (int(videoCallbackBuffer[2]) << 16) | (int(videoCallbackBuffer[3]) << 8) | int(videoCallbackBuffer[4])

						if pktType == 0 {
							b, err := libavc.ExtraDataToAnnexB(videoCallbackBuffer[5:])
							//if err != nil {
							//	return utils.AVCodecIdNONE, 0, err
							//}
							//d.videoExtraData = b
							println(b)
							println(err)
							println(ct)
							extraData = b
						} else if pktType == 1 {
							buffer := utils.NewByteBuffer()
							libavc.Mp4ToAnnexB(buffer, videoCallbackBuffer[5:], extraData)
							buffer.ReadTo(func(bytes []byte) {
								h264File.Write(bytes)
							})
							//return utils.AVCodecIdH264, ct, nil
						} else if pktType == 2 {
							//empty
						}

					} else {

					}
					videoBufferLength = 0
					videoLastPts += ts
				}

				copy(videoCallbackBuffer[videoBufferLength:], data)
				videoBufferLength += len(data)*/

	}, func(data []byte, ts int) {
		audioTS += ts
		//println(fmt.Sprintf("audio data lenth:%d ts:%d", len(data), audioTS))
		/*		soundFormat := data[0] >> 4
				//soundRate := data[0] >> 2 & 3
				//soundSize := data[0] >> 1 & 0x1
				//soundType := data[0] & 0x1
				soundData := data[1:]

				//aac audio data
				if soundFormat == 10 {
					//audio specificConfig
					if soundData[0] == 0x0 {
						audioConfig, err = utils.ParseMpeg4AudioConfig(data[2:])
						if err != nil {
							return
						}
					} else if soundData[0] == 0x1 {
						utils.SetADtsHeader(aacADtsHeader, 0, audioConfig.ObjectType-1, audioConfig.SamplingIndex, audioConfig.ChanConfig, 7+len(soundData[1:]))
						h264File.Write(aacADtsHeader)
						h264File.Write(soundData[1:])
						return
					}
				}*/
	})

	err = puller.Open(url)
	if err != nil {
		panic(err)
	}

	select {}
}
