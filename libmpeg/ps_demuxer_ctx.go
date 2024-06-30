package libmpeg

import (
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/libhevc"
	"github.com/yangjiechina/avformat/utils"
)

// PSDeMuxerContext 处理PSDeMuxer解析的部分es包，回调通知解析成完整包
// 由于不对es做任何缓存，故不解析AVStream
type PSDeMuxerContext struct {
	probeBuffer *PSProbeBuffer
	handler     Handler

	esCount     int //本次累计解析es长度
	esTotalSize int //es总大小
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

func NewPSDeMuxerContext(probeBuffer []byte) *PSDeMuxerContext {
	context := &PSDeMuxerContext{}

	muxer := NewPSDeMuxer()
	muxer.SetHandler(context.onEsPacket)
	context.probeBuffer = NewProbeBuffer(muxer, probeBuffer)
	return context
}

func (source *PSDeMuxerContext) Input(data []byte) error {
	return source.probeBuffer.Input(data)
}

func (source *PSDeMuxerContext) SetHandler(handler Handler) {
	source.handler = handler
}

func (source *PSDeMuxerContext) TrackCount() int {
	return len(source.probeBuffer.deMuxer.programStreamMap.elementaryStreams)
}

func (source *PSDeMuxerContext) findStreamIndex(mediaType utils.AVMediaType) int {
	streamIndex := -1

	for i, v := range source.streamIndexes {
		if v == mediaType {
			streamIndex = i
		}
	}

	if streamIndex == -1 {
		source.streamIndexes = append(source.streamIndexes, mediaType)
		streamIndex = len(source.streamIndexes) - 1
	}

	return streamIndex
}

// onEsPacket 从ps流中解析出来的es流回调
func (source *PSDeMuxerContext) onEsPacket(data []byte, total int, first bool, mediaType utils.AVMediaType, id utils.AVCodecID, dts int64, pts int64, params interface{}) error {
	length := len(data)

	//首包,根据pts和dts的变化, 组合完整的一帧
	if first && source.esCount > 0 && ((dts > source.dts || pts > source.pts) || source.mediaType != mediaType) {
		//丢包造成数据不足, 释放之前的缓存数据, 丢弃帧
		if source.esCount < source.esTotalSize {
			source.handler.OnLossPacket(source.findStreamIndex(source.mediaType), source.mediaType, source.codecId)
		} else {
			if err := source.handler.OnCompletePacket(source.findStreamIndex(source.mediaType), source.mediaType, source.codecId, source.dts, source.pts, source.idrFrame); err != nil {
				return err
			}
		}

		source.esCount = 0
	}

	if source.esCount == 0 {
		source.mediaType = mediaType
		source.codecId = id
		source.esTotalSize = total
		source.dts = dts
		source.pts = pts
		source.idrFrame = false
	}

	source.handler.OnPartPacket(source.findStreamIndex(mediaType), mediaType, id, data, source.esCount == 0)
	source.esCount += length

	//判断是否是关键帧, 不用等结束时循环判断.
	if first && utils.AVMediaTypeVideo == mediaType && !source.idrFrame {
		if utils.AVCodecIdH264 == id {
			source.idrFrame = libavc.H264NalIDRSlice == libavc.RemoveStartCode(data)[0]&0x1F
		} else if utils.AVCodecIdH265 == id {
			source.idrFrame = byte(libhevc.HevcNalIdrWRADL) == libavc.RemoveStartCode(data)[0]>>1&0x3F
		}
	}

	return nil
}

func (source *PSDeMuxerContext) Close() {
	source.handler = nil
	source.probeBuffer.deMuxer.Close()
}
