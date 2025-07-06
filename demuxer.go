package avformat

import (
	"fmt"
	"github.com/lkmio/avformat/bufio"
	"github.com/lkmio/avformat/collections"
	"github.com/lkmio/avformat/utils"
	"sort"
)

type Demuxer interface {
	Input(data []byte) (int, error)

	SetHandler(handler OnUnpackStreamHandler)

	DiscardBackPacket(index int)

	DiscardHeadPacket(index int)

	SetOnPreprocessPacketHandler(onPreprocessPacket func(packet *AVPacket))

	Close()

	SetProbeDuration(duration int)

	ProbeComplete()
}

type BaseDemuxer struct {
	Tracks                  TrackManager
	Handler                 OnUnpackStreamHandler
	DataPipeline            DataPipeline
	Name                    string // flv/ps/ts/rtp...
	OrderedStreams          []int
	ProbeDuration           int
	Completed               bool                                 // track解析完毕
	Packets                 []*collections.LinkedList[*AVPacket] // 保存在探测阶段解析出来的AVPacket
	AutoFree                bool                                 // 回调Packet后, 是否自动释放Packet
	streamIndex2BufferIndex map[int]int
	onPreprocessPacket      func(packet *AVPacket)
}

func (s *BaseDemuxer) Input(data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s *BaseDemuxer) SetHandler(handler OnUnpackStreamHandler) {
	s.Handler = handler
}

func (s *BaseDemuxer) GetTimebase() int {
	switch s.Name {
	case "flv", "jt1078":
		return 1000
	case "ps", "ts":
		return 90000
	default:
		panic(fmt.Sprintf("unknown demuxer name: %s", s.Name))
	}
}

func (s *BaseDemuxer) GetPackType() PacketType {

	switch s.Name {
	case "flv":
		return PacketTypeAVCC
	case "ps", "ts", "jt1078":
		return PacketTypeAnnexB
	default:
		return PacketTypeNONE
	}
}

func (s *BaseDemuxer) AddTrack(track Track) bool {
	return !s.Completed && s.Tracks.Add(track)
}

func (s *BaseDemuxer) OnNewAudioTrack(bufferIndex int, id utils.AVCodecID, timebase int, extraData []byte, config AudioConfig) Track {
	var track Track
	if s.Tracks.FindTrackWithType(utils.AVMediaTypeAudio) == nil || "ts" == s.Name {
		track = s.createAudioTrack(bufferIndex, id, timebase, extraData, config)
	}

	// 编码器信息会拷贝一份, 所以这里可以释放掉
	if len(extraData) > 0 {
		s.DataPipeline.DiscardBackPacket(bufferIndex)
	}
	return track
}

func (s *BaseDemuxer) createAudioTrack(bufferIndex int, id utils.AVCodecID, timebase int, extraData []byte, config AudioConfig) Track {
	stream := &AVStream{
		MediaType:   utils.AVMediaTypeAudio,
		Index:       s.Tracks.Size(),
		CodecID:     id,
		Timebase:    timebase,
		AudioConfig: config,
	}

	if length := len(extraData); length > 0 {
		data := make([]byte, length)
		copy(data, extraData)
		stream.Data = data
	}

	if utils.AVCodecIdAAC == id && !config.HasADTSHeader {
		utils.Assert(extraData != nil)
		mpeg4AudioConfig, err := utils.ParseMpeg4AudioConfig(extraData)
		if err != nil {
			println(err.Error())
			return nil
		}

		//, config.SampleRate, config.Channels
		stream.AudioConfig.SampleRate = mpeg4AudioConfig.SampleRate
		stream.AudioConfig.Channels = mpeg4AudioConfig.Channels
	}

	stream.Timebase = timebase
	track := &SimpleTrack{stream}
	if s.AddTrack(track) {
		if s.streamIndex2BufferIndex == nil {
			s.streamIndex2BufferIndex = make(map[int]int)
		}
		s.streamIndex2BufferIndex[track.GetStream().Index] = bufferIndex
		return track
	}

	println(fmt.Sprintf("failed to add audio track"))
	return nil
}

func (s *BaseDemuxer) OnNewVideoTrack(bufferIndex int, id utils.AVCodecID, timebase int, extraData []byte) Track {
	var track Track
	if s.Tracks.FindTrackWithType(utils.AVMediaTypeVideo) == nil || "ts" == s.Name {
		track = s.createVideoTrack(bufferIndex, id, timebase, extraData)
	}

	if len(extraData) > 0 {
		s.DataPipeline.DiscardBackPacket(bufferIndex)
	}
	return track
}

func (s *BaseDemuxer) createVideoTrack(bufferIndex int, id utils.AVCodecID, timebase int, extraData []byte) Track {
	data := make([]byte, len(extraData))
	copy(data, extraData)

	config, err := generateVideoCodecData(s.GetPackType(), id, data)
	if err != nil {
		println(err.Error())
		return nil
	}

	stream := NewAVStream(utils.AVMediaTypeVideo, s.Tracks.Size(), id, data, config)
	stream.Timebase = timebase
	track := &SimpleTrack{stream}
	if s.AddTrack(track) {
		if s.streamIndex2BufferIndex == nil {
			s.streamIndex2BufferIndex = make(map[int]int)
		}
		s.streamIndex2BufferIndex[track.GetStream().Index] = bufferIndex
		return track
	}

	println(fmt.Sprintf("failed to add video track"))
	return nil
}

func (s *BaseDemuxer) OnAudioPacket(bufferIndex int, id utils.AVCodecID, data []byte, ts int64) {
	var ok bool
	defer func() {
		if !ok {
			s.DataPipeline.DiscardBackPacket(bufferIndex)
		}
	}()

	// 如果track不存在, 封装AVStream
	track := s.findTrackByBufferIndex(bufferIndex)
	if !s.Completed && track == nil {
		extraData, _, config, err := ExtractAudioExtraData(id, data)
		if err != nil {
			println(err.Error())
			return
		}

		track = s.createAudioTrack(bufferIndex, id, s.GetTimebase(), extraData, config)
	}

	if track == nil {
		return
	}

	packet, err := ExtractAudioPacket(id, data, ts, track.GetStream().Index, track.GetStream().Timebase, track.GetStream().HasADTSHeader)
	if err != nil {
		println(err.Error())
		return
	}

	ok = true
	packet.BufferIndex = bufferIndex
	s.processBufferedPacket(packet)
}

func (s *BaseDemuxer) OnVideoPacket(bufferIndex int, id utils.AVCodecID, data []byte, key bool, dts, pts int64, packType PacketType) {
	var ok bool
	defer func() {
		if !ok {
			s.DataPipeline.DiscardBackPacket(bufferIndex)
		}
	}()

	// 如果track不存在, 并且是AnnexB打包, 从NALU中提取sps和pps封装AVStream
	track := s.findTrackByBufferIndex(bufferIndex)
	if track == nil && !s.Completed && PacketTypeAnnexB == s.GetPackType() {
		var extraData []byte
		var err error
		if key {
			extraData, err = ExtractVideoExtraDataFromKeyFrame(id, data)
			if err != nil {
				println(err.Error())
			}
		}

		// 如果没有找到, 则从旧的track中获取
		if extraData == nil {
			if oldTrack := s.Tracks.Find(id); oldTrack != nil {
				extraData = oldTrack.GetStream().Data
			}
		}

		if extraData != nil {
			track = s.createVideoTrack(bufferIndex, id, s.GetTimebase(), extraData)
		}
	}

	if track == nil {
		return
	}

	packet := NewVideoPacket(data, dts, pts, key, s.GetPackType(), id, track.GetStream().Index, track.GetStream().Timebase)

	ok = true
	packet.BufferIndex = bufferIndex
	if s.onPreprocessPacket != nil {
		s.onPreprocessPacket(packet)
	}
	s.processBufferedPacket(packet)
}

// 回调AVPacket, 如果没有完成探测, 则保存到Packets中, 否则回调处理
// 保证回调的顺序是OnNewTrack...->OnTrackComplete->OnPacket...
func (s *BaseDemuxer) processBufferedPacket(packet *AVPacket) {
	var packets *collections.LinkedList[*AVPacket]
	exist := packet.Index < len(s.Packets)
	if exist {
		packets = s.Packets[packet.Index]
	} else {
		s.Packets = append(s.Packets, &collections.LinkedList[*AVPacket]{})
		packets = s.Packets[len(s.Packets)-1]
	}

	packets.Add(packet)

	prevPacketIndex := packets.Size() - 2
	if prevPacketIndex < 0 {
		return
	}

	// 计算上一个AVPacket的duration
	prevPacket := packets.Get(prevPacketIndex)
	prevPacket.Duration = packet.Dts - prevPacket.Dts

	// 如果已经完成探测, 则回调处理
	if s.Completed {
		s.forwardPacket(packets.Remove(prevPacketIndex))
		return
	}

	// 探测时长
	var duration int
	for i := 0; i <= prevPacketIndex; i++ {
		duration += int(packets.Get(i).GetDuration(1000))
	}

	if duration >= bufio.MaxInt(bufio.MaxInt(s.ProbeDuration, 200), duration) {
		s.OnProbeComplete()
	}
}

func (s *BaseDemuxer) forwardPacket(packet *AVPacket) {
	s.Handler.OnPacket(packet)
	if s.AutoFree {
		// 释放内存
		bufferIndex, ok := s.streamIndex2BufferIndex[packet.Index]
		utils.Assert(ok)
		s.DataPipeline.DiscardHeadPacket(bufferIndex)
		FreePacket(packet)
	}
}

func (s *BaseDemuxer) DiscardBackPacket(index int) {
	s.DataPipeline.DiscardBackPacket(index)
}

func (s *BaseDemuxer) DiscardHeadPacket(index int) {
	s.DataPipeline.DiscardHeadPacket(index)
}

func (s *BaseDemuxer) FindBufferIndexByMediaType(mediaType utils.AVMediaType) int {
	return s.FindBufferIndex(int(mediaType))
}

func (s *BaseDemuxer) FindBufferIndex(marker int) int {
	for i, stream := range s.OrderedStreams {
		if marker == stream {
			return i
		}
	}

	s.OrderedStreams = append(s.OrderedStreams, marker)
	return len(s.OrderedStreams) - 1
}

func (s *BaseDemuxer) findTrackByBufferIndex(index int) Track {
	for streami, bufferi := range s.streamIndex2BufferIndex {
		if index == bufferi {
			return s.Tracks.Get(streami)
		}
	}

	return nil
}

func (s *BaseDemuxer) OnProbeComplete() {
	if s.Completed {
		return
	}

	s.Completed = true
	if s.Tracks.Size() == 0 {
		s.Handler.OnTrackNotFind()
		return
	}

	for _, track := range s.Tracks.Tracks {
		s.Handler.OnNewTrack(track)
	}

	s.Handler.OnTrackComplete()

	// 回调之前保存的AVPacket
	var result []*AVPacket
	for _, packets := range s.Packets {
		prePacketIndex := packets.Size() - 2
		if prePacketIndex < 0 {
			continue
		}

		for i := 0; i <= prePacketIndex; i++ {
			packet := packets.Remove(0)
			//s.forwardPacket(packet)
			result = append(result, packet)
		}
	}

	// dts升序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Dts < result[j].Dts
	})

	for _, packet := range result {
		s.forwardPacket(packet)
	}
}

func (s *BaseDemuxer) SetOnPreprocessPacketHandler(onPreprocessPacket func(packet *AVPacket)) {
	s.onPreprocessPacket = onPreprocessPacket
}

func (s *BaseDemuxer) Close() {
	s.Handler = nil
	s.onPreprocessPacket = nil
}

func (s *BaseDemuxer) SetProbeDuration(duration int) {
	s.ProbeDuration = duration
}

func (s *BaseDemuxer) ProbeComplete() {
	s.OnProbeComplete()
}
