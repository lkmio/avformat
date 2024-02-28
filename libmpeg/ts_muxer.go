package libmpeg

import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
	"math"
)

/**
 * TS视频流，由多个固定大小的TS包组成, 一般多为188包长. 每个TS包都拥有一个TSHeader, 用于标识TS流数据类型和序号.
 * TS包分为2种，PSI和PES. PSI(PAT/PMT...)用于描述音视频流信息. PES负载音视频流.
 * PAT携带PMT的PID, PMT里面存储音视频的StreamType和StreamId
 * PAT->PMT->DATA...PAT->PMT->DATA
 */
type TSMuxer interface {
	AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID) (int, error)

	// WriteHeader 写入PAT和PMT /**
	WriteHeader()

	Input(trackIndex int, data []byte, dts, pts int64) error

	// Reset 清空tracks, 可重新AddTrack和WriteHeader /**
	Reset()

	SetAllocHandler(func(size int) []byte)

	SetWriteHandler(func(data []byte))
}

func NewTSMuxer() TSMuxer {
	return &tsMuxer{}
}

type tsTrack struct {
	streamType int
	pes        *PESHeader
	buffer     []byte
	tsHeader   *TSHeader
	mediaType  utils.AVMediaType
	avCodecId  utils.AVCodecID
}

type tsMuxer struct {
	tracks []*tsTrack

	allocHandler func(size int) []byte
	writeHandler func([]byte)
}

func (t *tsMuxer) AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID) (int, error) {
	var pes *PESHeader
	if utils.AVMediaTypeAudio == mediaType {
		pes = NewPESPacket(StreamIdAudio)
	} else if utils.AVMediaTypeVideo == mediaType {
		pes = NewPESPacket(StreamIdVideo)
	} else {
		utils.Assert(false)
	}

	streamType, ok := codecId2StreamTypeMap[id]
	if !ok {
		return -1, fmt.Errorf("the codec %d does not support mux to TS stream", int(id))
	}

	for _, track := range t.tracks {
		utils.Assert(track.streamType != streamType)
	}

	tsHeader := NewTSHeader(TsPacketStartPid+len(t.tracks), 1, 0)
	t.tracks = append(t.tracks, &tsTrack{streamType: streamType, pes: pes, buffer: make([]byte, 128), tsHeader: &tsHeader,
		mediaType: mediaType, avCodecId: id})
	return len(t.tracks) - 1, nil
}

func (t *tsMuxer) WriteHeader() {
	utils.Assert(len(t.tracks) > 0)

	//写PAT
	bytes := t.allocHandler(TsPacketSize * 2)
	n := writePAT(bytes, 0)
	utils.Assert(n > 0 && n < TsPacketSize)
	copy(bytes[n:], stuffing[n:])

	//写PMT
	streamTypes := make([][2]int16, len(t.tracks))
	for index, track := range t.tracks {
		streamTypes[index][0] = int16(track.streamType)
		streamTypes[index][1] = int16(track.tsHeader.pid)
	}

	n = writePMT(bytes[TsPacketSize:], 0, 1, TsPacketStartPid, streamTypes)
	utils.Assert(n > 0 && n < TsPacketSize)
	copy(bytes[TsPacketSize+n:], stuffing[:TsPacketSize-n])

	t.writeHandler(bytes)
}

func (t *tsMuxer) Input(trackIndex int, data []byte, dts, pts int64) error {
	track := t.tracks[trackIndex]

	//不固定PES包长
	track.pes.packetLength = 0x0000
	track.pes.ptsDtsFlags = PesExistPtsMark
	track.pes.dts = dts
	track.pes.pts = pts

	//给音视频帧加上pes头, 再分割成多个TS包
	pesHeaderLen := track.pes.ToBytes(track.buffer)
	pesLen := len(data) + pesHeaderLen

	for remain := pesLen; remain > 0; {
		bytes := t.allocHandler(TsPacketSize)
		pktSize := TsPacketSize - 4

		//首包加入pcr和aud
		if remain == pesLen {
			pktSize -= track.tsHeader.writePCR(bytes[4:], pts*300)
			track.tsHeader.payloadUnitStartIndicator = 1
		} else {
			track.tsHeader.payloadUnitStartIndicator = 0
		}

		//不足一个包, 填充满188个字节
		fillCount := pktSize - remain
		if fillCount > 0 {
			fillCount -= 2
			pktSize -= track.tsHeader.fill(bytes[TsPacketSize-pktSize:], int(math.Abs(float64(fillCount))), remain != pesLen)

			//填充的数量 必须要大于自适应字段的最低长度, 否则不够写. 只能少些点pes数据
			if fillCount < 0 {
				pktSize += fillCount
			}
		}

		//拷贝pes头
		if pesHeaderLen > 0 {
			utils.Assert(pktSize > pesHeaderLen)

			copy(bytes[TsPacketSize-pktSize:], track.buffer[:pesHeaderLen])

			pktSize -= pesHeaderLen
			remain -= pesHeaderLen
			pesLen = remain
			pesHeaderLen = 0

			//ios平台/vlc播放需要添加aud
			//传入的nalu不要携带aud
			if utils.AVMediaTypeVideo == track.mediaType {
				utils.Assert(pktSize > 6)
				pktSize -= writeAud(bytes[TsPacketSize-pktSize:], track.avCodecId)
			}
		}

		copy(bytes[TsPacketSize-pktSize:], data[pesLen-remain:pesLen-remain+pktSize])
		remain -= pktSize
		track.tsHeader.toBytes(bytes[:])
		t.writeHandler(bytes[:TsPacketSize])
		track.tsHeader.increaseCounter()
	}

	return nil
}

func (t *tsMuxer) Reset() {

}

func (t *tsMuxer) SetAllocHandler(f func(size int) []byte) {
	t.allocHandler = f
}

func (t *tsMuxer) SetWriteHandler(f func(data []byte)) {
	t.writeHandler = f
}
