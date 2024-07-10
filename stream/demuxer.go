package stream

import "github.com/lkmio/avformat/utils"

// OnDeMuxerHandler 解复用器回调
type OnDeMuxerHandler interface {
	OnDeMuxStream(stream utils.AVStream)
	OnDeMuxStreamDone()
	OnDeMuxPacket(packet utils.AVPacket)
	OnDeMuxDone()
}

// DeMuxer 解复用器接口
type DeMuxer interface {
	// Input 输入需要解复用的数据
	// @param vod 调用者和解复用器之间传递的私有数据
	Input(data []byte /*, vod interface{}*/) (int, error)

	SetHandler(handler OnDeMuxerHandler)

	Close()
}

type DeMuxerImpl struct {
	Handler OnDeMuxerHandler
}

func (deMuxer *DeMuxerImpl) SetHandler(handler OnDeMuxerHandler) {
	deMuxer.Handler = handler
}

func (deMuxer *DeMuxerImpl) Close() {
	deMuxer.Handler = nil
}

// OnTransDeMuxerHandler 从传输协议层中(RTMP/RTP...)解析出AVPacket/AVStream, 这个过程是有序的，可以和GOP缓存绑定，减少内存拷贝
// DeMuxer在解析到音视频数据，需要拷贝时，通过OnPartPacket回调出去，外部拷贝. 解析到完整帧时，依旧使用OnDeMuxStream/OnDeMuxPacket通知
type OnTransDeMuxerHandler interface {
	// OnPartPacket
	// @param index 和stream index类似
	// @param first 是否是第一包数据, 可以区分是否完整帧
	OnPartPacket(index int, data []byte, first bool)
}
