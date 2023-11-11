package avformat

import "github.com/yangjiechina/avformat/utils"

// OnDeMuxerHandler 解复用器回调
type OnDeMuxerHandler interface {
	OnDeMuxStream(stream utils.AVStream)
	OnDeMuxStreamDone()
	OnDeMuxPacket(index int, packet *utils.AVPacket2)
	OnDeMuxDone()
}

// DeMuxer 解复用器接口
type DeMuxer interface {
	// Input 输入需要解复用的数据
	// vod 调用者和解复用器之间传递的私有数据
	Input(data []byte, vod interface{})

	SetHandler(handler OnDeMuxerHandler)
}

type DeMuxerImpl struct {
	Handler OnDeMuxerHandler
}

func (deMuxer *DeMuxerImpl) SetHandler(handler OnDeMuxerHandler) {
	deMuxer.Handler = handler
}
