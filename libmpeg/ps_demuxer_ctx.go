package libmpeg

import (
	"github.com/lkmio/avformat/libavc"
	"github.com/lkmio/avformat/libhevc"
	"github.com/lkmio/avformat/utils"
)

// PSDeMuxerContext 处理PSDeMuxer解析的部分es包，回调通知解析成完整包
// 由于不对es做任何缓存，故不解析AVStream
type PSDeMuxerContext struct {
	probeBuffer *PSProbeBuffer
	handler     Handler

	esCount     int // 本次累计的es长度
	esTotalSize int // es总大小
	idrFrame    bool

	pts       int64
	dts       int64
	mediaType utils.AVMediaType
	codecId   utils.AVCodecID

	streamIndexes []utils.AVMediaType
}

type Handler interface {
	OnPartPacket(streamIndex int, mediaType utils.AVMediaType, codec utils.AVCodecID, data []byte, first bool)

	OnLossPacket(streamIndex int, mediaType utils.AVMediaType, codec utils.AVCodecID)

	OnCompletePacket(streamIndex int, mediaType utils.AVMediaType, codec utils.AVCodecID, dts int64, pts int64, key bool) error
}

func (ctx *PSDeMuxerContext) Input(data []byte) error {
	return ctx.probeBuffer.Input(data)
}

func (ctx *PSDeMuxerContext) SetHandler(handler Handler) {
	ctx.handler = handler
}

func (ctx *PSDeMuxerContext) TrackCount() int {
	return len(ctx.probeBuffer.deMuxer.programStreamMap.elementaryStreams)
}

func (ctx *PSDeMuxerContext) findStreamIndex(mediaType utils.AVMediaType) int {
	streamIndex := -1

	for i, v := range ctx.streamIndexes {
		if v == mediaType {
			streamIndex = i
		}
	}

	if streamIndex == -1 {
		ctx.streamIndexes = append(ctx.streamIndexes, mediaType)
		streamIndex = len(ctx.streamIndexes) - 1
	}

	return streamIndex
}

var count int

// onEsPacket 从ps流中解析出来的es流回调
func (ctx *PSDeMuxerContext) onEsPacket(data []byte, total int, first bool, mediaType utils.AVMediaType, id utils.AVCodecID, dts int64, pts int64) error {
	length := len(data)
	count++
	// 首包, 处理前一个es包. 根据时间戳和类型的变化, 决定丢弃或者回调完整包
	if first && ctx.esCount > 0 && ((dts > ctx.dts || pts > ctx.pts) || ctx.mediaType != mediaType) {
		tmp := ctx.esCount
		ctx.esCount = 0
		// 丢包造成数据不足, 释放之前的缓存数据, 丢弃帧
		if tmp < ctx.esTotalSize {
			ctx.handler.OnLossPacket(ctx.findStreamIndex(ctx.mediaType), ctx.mediaType, ctx.codecId)
		} else {
			if err := ctx.handler.OnCompletePacket(ctx.findStreamIndex(ctx.mediaType), ctx.mediaType, ctx.codecId, ctx.dts, ctx.pts, ctx.idrFrame); err != nil {
				return err
			}
		}
	}

	// 第一包, 重置标记
	if ctx.esCount == 0 {
		ctx.mediaType = mediaType
		ctx.codecId = id
		ctx.esTotalSize = total
		ctx.dts = dts
		ctx.pts = pts
		ctx.idrFrame = false
	}

	// 回调部分es数据
	ctx.handler.OnPartPacket(ctx.findStreamIndex(mediaType), mediaType, id, data, ctx.esCount == 0)
	ctx.esCount += length

	// 判断是否是关键帧, 不用等结束时循环判断.
	if first && utils.AVMediaTypeVideo == mediaType && !ctx.idrFrame {
		if utils.AVCodecIdH264 == id {
			ctx.idrFrame = libavc.IsKeyFrame(data)
		} else if utils.AVCodecIdH265 == id {
			ctx.idrFrame = libhevc.IsKeyFrame(data)
		}
	}

	return nil
}

func (ctx *PSDeMuxerContext) Close() {
	ctx.handler = nil
	ctx.probeBuffer.deMuxer.Close()
}

func NewPSDeMuxerContext(probeBuffer []byte) *PSDeMuxerContext {
	context := &PSDeMuxerContext{}

	muxer := NewPSDeMuxer()
	muxer.SetHandler(context.onEsPacket)
	context.probeBuffer = NewProbeBuffer(muxer, probeBuffer)
	return context
}
