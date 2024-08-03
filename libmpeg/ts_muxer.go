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
	buffer           []byte
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
		} else if utils.AVCodecIdH264 == id {
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

	track.pes.ptsDtsFlags = PesExistPtsMark
	track.pes.dts = dts
	track.pes.pts = pts

	//给音视频帧加上pes头, 再分割成多个TS包
	pesHeaderLen := track.pes.ToBytes(track.buffer)
	pesLen := size + pesHeaderLen

	for remain := pesLen; remain > 0; {
		bytes := t.allocHandler(TsPacketSize)
		pktSize := TsPacketSize - 4

		//首包加入pcr
		if remain == pesLen {
			if utils.AVMediaTypeVideo == track.mediaType && t.startTS == pts {
				pktSize -= track.tsHeader.writePCR(bytes[4:], pts*300)
			}
			track.tsHeader.payloadUnitStartIndicator = 1
		} else {
			track.tsHeader.payloadUnitStartIndicator = 0
		}

		//不足一个包, 填充满188个字节
		fillCount := pktSize - remain
		if fillCount > 0 {
			pktSize -= track.tsHeader.fill(bytes[TsPacketSize-pktSize:], fillCount)
		}

		//拷贝pes头
		if pesHeaderLen > 0 {
			utils.Assert(pktSize > pesHeaderLen)
			packetLength := pesLen - 6
			pesStartIndex := TsPacketSize - pktSize + 4

			copy(bytes[TsPacketSize-pktSize:], track.buffer[:pesHeaderLen])

			pktSize -= pesHeaderLen
			remain -= pesHeaderLen
			pesLen = remain
			pesHeaderLen = 0

			//ios平台/vlc播放需要添加aud
			//传入的nalu不要携带aud
			if utils.AVMediaTypeVideo == track.mediaType {
				utils.Assert(pktSize > 6)
				audLength := writeAud(bytes[TsPacketSize-pktSize:], track.codecId)
				packetLength += audLength
				pktSize -= audLength
			}

			if packetLength <= 65535 {
				bytes[pesStartIndex] = byte(packetLength >> 8 & 0xFF)
				bytes[pesStartIndex+1] = byte(packetLength & 0xFF)
			} else {
				bytes[pesStartIndex] = 0x00
				bytes[pesStartIndex+1] = 0x00
			}
		}

		index := pesLen - remain
		for _, pkt := range data {
			if pktSize == 0 {
				break
			}

			if index < len(pkt) {
				remainCount := len(pkt[index:])
				minInt := libbufio.MinInt(remainCount, pktSize)
				copy(bytes[TsPacketSize-pktSize:], pkt[index:index+minInt])
				remain -= minInt
				pktSize -= minInt
			} else {
				index -= len(pkt)
			}
		}

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
	} else if utils.AVCodecIdH264 == track.codecId && key && track.extra != nil {
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
