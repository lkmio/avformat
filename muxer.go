package avformat

import "github.com/yangjiechina/avformat/utils"

// Muxer 复用器
type Muxer interface {

	// AddStream 添加Track
	AddStream(type_ utils.AVMediaType, codecId utils.AVCodecID, extra []byte) int

	// Input 向track写入数据
	Input(trackIndex int, packet utils.AVPacket)

	// WriteHeader 写文件头，添加所有track后调用
	WriteHeader()

	// Close 关闭复用器，同时写文件尾
	Close()
}
