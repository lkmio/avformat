package libmpeg

import (
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
)

/**
 * TS视频流，由多个固定大小的TS包组成, 一般多为188包长. 每个TS包都拥有一个TSHeader, 用于标识TS流数据类型和序号.
 * TS包分为2种，PSI和PES. PSI(PAT/PMT...)用于描述音视频流信息. PES负载音视频流.
 * PAT携带PMT的PID, PMT里面存储音视频的StreamType和StreamId
 * PAT->PMT->DATA...PAT->PMT->DATA
 */
type TSMuxer interface {
	AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID, extra []byte) (int, error)

	// WriteHeader 写入PAT和PMT /**
	WriteHeader() error

	Input(trackIndex int, data []byte, pts, dts int64, key bool) error

	// Reset 清空tracks, 可重新AddTrack和WriteHeader /**
	Reset()

	SetAllocHandler(func(size int) []byte)

	SetWriteHandler(func(data []byte))

	TrackCount() int

	Duration() int64

	Close()
}

func NewTSMuxer() TSMuxer {
	return &tsMuxer{
		startTS: -1,
		endTS:   -1,
	}
}

type tsTrack struct {
	streamType       int
	pes              *PESHeader
	buffer           []byte // 临时保存pes头
	tsHeader         *TSHeader
	mediaType        utils.AVMediaType
	codecId          utils.AVCodecID
	extra            []byte
	extraConfig      interface{}
	extraWriteBuffer []byte
}

type tsMuxer struct {
	tracks  []*tsTrack
	startTS int64
	endTS   int64

	allocHandler func(size int) []byte
	writeHandler func([]byte)
}

func (t *tsMuxer) AddTrack(mediaType utils.AVMediaType, id utils.AVCodecID, extra []byte) (int, error) {
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
	track := &tsTrack{streamType: streamType, pes: pes, buffer: make([]byte, 128), tsHeader: &tsHeader,
		mediaType: mediaType, codecId: id}
	t.tracks = append(t.tracks, track)

	if extra != nil {
		if utils.AVCodecIdAAC == id {
			record, err := utils.ParseMpeg4AudioConfig(extra)
			if err != nil {
				return 0, err
			}
			track.extraConfig = record
			//adts header
			track.extraWriteBuffer = make([]byte, 7)
		} else {
			track.extra = extra
		}
	}

	return len(t.tracks) - 1, nil
}

func (t *tsMuxer) WriteHeader() error {
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
	return nil
}

func (t *tsMuxer) write(track *tsTrack, pts, dts int64, data ...[]byte) error {
	size := 0
	for _, bytes := range data {
		size += len(bytes)
	}

	track.pes.ptsDtsFlags = PesExistPtsDtsMark
	track.pes.dts = dts
	track.pes.pts = pts

	// 每帧加一个pes头, 再分割成多个TS包
	pesHeaderLen := track.pes.ToBytes(track.buffer)
	// pes头+aud
	headerSize := pesHeaderLen
	// 视频帧还需要加上aud头的长度
	if utils.AVMediaTypeVideo == track.mediaType {
		aud := [7]byte{}
		if n := writeAud(aud[:], track.codecId); n > 0 {
			headerSize += n
		}
	}

	for remain := size; remain > 0; {
		bytes := t.allocHandler(TsPacketSize)
		// -4预留ts包头
		tsPktSize := TsPacketSize - 4

		// 首包加入pcr
		if remain == size {
			if utils.AVMediaTypeVideo == track.mediaType && t.startTS == pts {
				tsPktSize -= track.tsHeader.writePCR(bytes[4:], pts*300)
			}
			track.tsHeader.payloadUnitStartIndicator = 1
		} else {
			track.tsHeader.payloadUnitStartIndicator = 0
		}

		// 不足188个字节, 填充0xFF
		fillCount := tsPktSize - headerSize - remain
		if fillCount > 0 {
			tsPktSize -= track.tsHeader.fill(bytes[TsPacketSize-tsPktSize:], fillCount)
		}

		// 拷贝pes头
		if pesHeaderLen > 0 {
			utils.Assert(tsPktSize > pesHeaderLen)
			lengthOffset := TsPacketSize - tsPktSize + 4
			copy(bytes[TsPacketSize-tsPktSize:], track.buffer[:pesHeaderLen])
			tsPktSize -= pesHeaderLen
			pesHeaderLen = 0

			// ios平台/vlc播放需要添加aud
			// 传入的nalu不要携带aud
			if utils.AVMediaTypeVideo == track.mediaType {
				utils.Assert(tsPktSize > 6)
				tsPktSize -= writeAud(bytes[TsPacketSize-tsPktSize:], track.codecId)
			}

			// 写pes包长度, 如果超过2字节, 设置为0(不定长)
			// 减去0x000001E0+长度(2bytes)
			pesPktLength := pesHeaderLen - 6 + headerSize + size
			if pesPktLength <= 65535 {
				bytes[lengthOffset] = byte(pesPktLength >> 8 & 0xFF)
				bytes[lengthOffset+1] = byte(pesPktLength & 0xFF)
			} else {
				bytes[lengthOffset] = 0x00
				bytes[lengthOffset+1] = 0x00
			}

			headerSize = 0
		}

		// 拷贝es数据
		index := size - remain
		for _, pkt := range data {
			if tsPktSize == 0 {
				break
			}

			if index < len(pkt) {
				remainCount := len(pkt[index:])
				n := libbufio.MinInt(remainCount, tsPktSize)
				copy(bytes[TsPacketSize-tsPktSize:], pkt[index:index+n])
				remain -= n
				tsPktSize -= n
			} else {
				index -= len(pkt)
			}
		}

		// 完整ts包
		utils.Assert(tsPktSize == 0)
		track.tsHeader.toBytes(bytes[:])
		t.writeHandler(bytes[:TsPacketSize])
		track.tsHeader.increaseCounter()
	}

	return nil
}

func (t *tsMuxer) Input(trackIndex int, data []byte, pts, dts int64, key bool) error {
	if t.startTS == -1 {
		t.startTS = pts
	}

	if pts < t.startTS {
		t.endTS = t.startTS
		t.startTS = pts
	} else {
		t.endTS = pts
	}

	pts = pts % 0x1FFFFFFFF
	dts = dts % 0x1FFFFFFFF
	track := t.tracks[trackIndex]
	if track.codecId == utils.AVCodecIdAAC && track.extraConfig != nil {
		audioConfig := track.extraConfig.(*utils.MPEG4AudioConfig)
		utils.SetADtsHeader(track.extraWriteBuffer, 0, audioConfig.ObjectType-1, audioConfig.SamplingIndex, audioConfig.ChanConfig, 7+len(data))
		return t.write(track, pts, dts, track.extraWriteBuffer, data)
	} else if utils.AVMediaTypeVideo == track.mediaType && key && track.extra != nil {
		return t.write(track, pts, dts, track.extra, data)
	}

	return t.write(track, pts, dts, data)
}

func (t *tsMuxer) Reset() {
	t.startTS = -1
	t.endTS = t.startTS

	for _, track := range t.tracks {
		track.tsHeader.payloadUnitStartIndicator = 1
		track.tsHeader.continuityCounter = 0
	}
}

func (t *tsMuxer) SetAllocHandler(f func(size int) []byte) {
	t.allocHandler = f
}

func (t *tsMuxer) SetWriteHandler(f func(data []byte)) {
	t.writeHandler = f
}

func (t *tsMuxer) TrackCount() int {
	return len(t.tracks)
}

func (t *tsMuxer) Duration() int64 {
	return t.endTS - t.startTS
}

func (t *tsMuxer) Close() {
	t.allocHandler = nil
	t.writeHandler = nil
}
