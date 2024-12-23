package libflv

import (
	"encoding/binary"
	"fmt"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/stream"
	"github.com/lkmio/avformat/utils"
)

type TagType byte
type VideoCodecId uint32
type SoundFormat byte
type SoundRate byte
type PacketType byte

const (
	TagTypeAudioData        = TagType(8)
	TagTypeVideoData        = TagType(9)
	TagTypeScriptDataObject = TagType(18) //metadata value https://en.wikipedia.org/wiki/Flash_Video

	VideoCodeIdH263     = VideoCodecId(2)
	VideoCodeIdSCREEN   = VideoCodecId(3)
	VideoCodeIdVP6      = VideoCodecId(4)
	VideoCodeIdVP6Alpha = VideoCodecId(5)
	VideoCodeIdScreenV2 = VideoCodecId(6)
	VideoCodeIdH264     = VideoCodecId(7)

	SoundFormatMP3      = SoundFormat(2)
	SoundFormatG711A    = SoundFormat(7)
	SoundFormatG711B    = SoundFormat(8)
	SoundFormatAAC      = SoundFormat(10)
	SoundFormatMP38K    = SoundFormat(14)
	SoundFormatExHeader = SoundFormat(9)

	SoundRate5500HZ  = SoundRate(0)
	SoundRate11000HZ = SoundRate(1)
	SoundRate22000HZ = SoundRate(2)
	SoundRate44000HZ = SoundRate(3) //For AAC:always 3

	AACFrameSize = 1024
	MP3FrameSize = 1152

	PacketTypeSequenceStart        = PacketType(0) //新的视频序列开始
	PacketTypeCodedFrames          = PacketType(1)
	PacketTypeSequenceEnd          = PacketType(2) //新的视频序列结束
	PacketTypeCodedFramesX         = PacketType(3) //不包含CompositionTime
	PacketTypeMetaData             = PacketType(4) //元数据,例如:h265中的HDR信息
	PacketTypeMPEG2TSSequenceStart = PacketType(5)

	AudioPacketTypeSequenceStart      = PacketType(0)
	AudioPacketTypeCodedFrames        = PacketType(1)
	AudioPacketTypeSequenceEnd        = PacketType(2)
	AudioPacketTypeMultichannelConfig = PacketType(4)
	AudioPacketTypeMultiTrack         = PacketType(5)

	TagHeaderSize = 15 //该长度包含preTagSize
)

var (
	VideoCodeIdAV1  = VideoCodecId(1635135537)
	VideoCodeIdVP9  = VideoCodecId(1987063865)
	VideoCodeIdHEVC = VideoCodecId(1752589105)
)

func init() {
	VideoCodeIdAV1 = VideoCodecId(makeFourCc("av01"))
	VideoCodeIdVP9 = VideoCodecId(makeFourCc("vp09"))
	VideoCodeIdHEVC = VideoCodecId(makeFourCc("hvc1"))
}

type DeMuxer interface {
	stream.DeMuxer

	// InputVideo 输入不带tag的视频帧
	InputVideo(data []byte, ts uint32) error

	// InputAudio 输入不带tag的音频帧
	InputAudio(data []byte, ts uint32) error
}

type deMuxer struct {
	stream.DeMuxerImpl

	metaData        *AMF0 // 元数据
	headerProcessed bool  // 是否已经读取到9字节的FLV头
	tag             Tag   // 保存当前正在读取的Tag

	audioIndex  int
	videoIndex  int
	audioTs     int64
	videoTs     int64
	videoStream utils.AVStream
	audioStream utils.AVStream
}

type Tag struct {
	preSize   uint32
	type_     TagType
	dataSize  int
	timestamp uint32

	data []byte
	size int
}

func (d *deMuxer) readScriptDataObject(data []byte) error {
	amf0 := &AMF0{}
	err := amf0.Unmarshal(data)
	if err != nil {
		return err
	}

	d.metaData = amf0
	return nil
}

func (d *deMuxer) readHeader(data []byte) error {
	if len(data) < 9 {
		return fmt.Errorf("the header of FLV requires 9 bytes")
	}

	if data[0] != 0x46 || data[1] != 0x4c || data[2] != 0x56 {
		return fmt.Errorf("the signature of FLV matching failed")
	}

	version := data[3]
	flags := typeFlag(data[4])
	dataOffset := binary.BigEndian.Uint32(data[5:])
	if version == 1 && dataOffset != 9 {
		return fmt.Errorf("invalid data")
	}

	if !flags.ExistAudio() && !flags.ExistVideo() {
		return fmt.Errorf("invalid data")
	}

	return nil
}

// 读取tag
// uint32 pre size
// TagType tag类型
// int data size
// uint32 timestamp
func (d *deMuxer) readTag(data []byte) Tag {
	_ = data[TagHeaderSize]
	timestamp := libbufio.Uint24(data[8:])
	timestamp |= uint32(data[11]) << 24

	return Tag{preSize: binary.BigEndian.Uint32(data), type_: TagType(data[4]), dataSize: int(libbufio.Uint24(data[5:])), timestamp: timestamp}
}

func (d *deMuxer) parseTag(data []byte, tagType TagType, ts uint32) error {
	if TagTypeAudioData == tagType {
		err := d.InputAudio(data, ts)
		if err != nil {
			return err
		}
	} else if TagTypeVideoData == tagType {
		err := d.InputVideo(data, ts)
		if err != nil {
			return err
		}
	} else if TagTypeScriptDataObject == tagType {
		if err := d.readScriptDataObject(data); err != nil {
			return err
		}
	}

	return nil
}

// Input 输入tag
func (d *deMuxer) Input(data []byte) (int, error) {
	var n int
	if !d.headerProcessed {
		if err := d.readHeader(data); err != nil {
			return -1, err
		}

		d.headerProcessed = true
		n = 9
	}

	// 读取未解析完的Tag
	need := d.tag.dataSize - d.tag.size
	if need > 0 {
		min := libbufio.MinInt(len(data), need)
		copy(d.tag.data[d.tag.size:], data[:min])
		d.tag.size += min
		n = min

		if min != need {
			return n, nil
		}

		err := d.parseTag(d.tag.data[:d.tag.size], d.tag.type_, d.tag.timestamp)
		if err != nil {
			return n, err
		}

		d.tag.size = 0
		d.tag.dataSize = 0
	}

	for len(data[n:]) > TagHeaderSize {
		tag := d.readTag(data[n:])
		n += TagHeaderSize

		//数据不够，保存起，等下次
		if len(data[n:]) < tag.dataSize {
			tmp := d.tag.data
			d.tag = tag
			d.tag.data = tmp

			if cap(d.tag.data) < tag.dataSize {
				d.tag.data = make([]byte, tag.dataSize)
			}

			copy(d.tag.data, data[n:])
			d.tag.size = len(data[n:])
			n = len(data)
			break
		}

		err := d.parseTag(data[n:n+tag.dataSize], tag.type_, tag.timestamp)
		if err != nil {
			return n, err
		}

		n += tag.dataSize
	}

	return n, nil
}

// InputVideo 输入不带tag的视频帧
func (d *deMuxer) InputVideo(data []byte, ts uint32) error {
	if d.videoIndex == -1 {
		d.videoIndex = d.audioIndex + 1
	}

	n, sequenceHeader, key, codecId, ct, err := ParseVideoData(data)
	if err != nil {
		return err
	} else if utils.AVCodecIdNONE == codecId {
		return nil
	}

	if sequenceHeader {
		if d.videoStream != nil {
			return nil
		}

		var config utils.CodecData
		extraData := make([]byte, len(data[n:]))
		copy(extraData, data[n:])

		if utils.AVCodecIdH264 == codecId {
			self, err := utils.ParseAVCDecoderConfigurationRecord(extraData)
			if err != nil {
				return err
			}

			config = self
		} else if utils.AVCodecIdH265 == codecId {
			self, err := utils.ParseHEVCDecoderConfigurationRecord(extraData)
			if err != nil {
				return err
			}

			config = self
		}

		d.videoStream = utils.NewAVStream(utils.AVMediaTypeVideo, d.videoIndex, codecId, extraData, config)
		d.Handler.OnDeMuxStream(d.videoStream)
		if d.audioIndex != -1 {
			d.Handler.OnDeMuxStreamDone()
		}

		return nil
	}

	if d.videoStream == nil {
		return fmt.Errorf("missing video sequence header")
	}

	var duration int64
	if d.videoTs != -1 {
		duration = int64(ts) - d.videoTs
	}

	// 时间戳溢出
	// ts是累加的, 除了溢出, 不会存在时间戳比前一个帧小的情况
	if d.videoTs != -1 && int64(ts) < d.videoTs {
		duration = 0xFFFFFFFF - d.videoTs + int64(ts)
	}

	d.videoTs = int64(ts)
	packet := utils.NewVideoPacket(data[n:], d.videoTs, d.videoTs+int64(ct), key, utils.PacketTypeAVCC, codecId, d.videoIndex, 1000)
	packet.SetDuration(duration)
	d.Handler.OnDeMuxPacket(packet)

	return nil
}

func (d *deMuxer) InputAudio(data []byte, ts uint32) error {
	if d.audioIndex == -1 {
		d.audioIndex = d.videoIndex + 1
	}

	n, sequenceHeader, codecId, err := ParseAudioData(data)
	if err != nil {
		return err
	}

	if d.audioStream == nil {
		if d.audioStream != nil {
			return nil
		}

		var audioStream utils.AVStream
		if utils.AVCodecIdAAC == codecId && sequenceHeader {
			extraData := make([]byte, len(data[n:]))
			copy(extraData, data[n:])
			config, err := utils.ParseMpeg4AudioConfig(extraData)
			if err != nil {
				return err
			}

			audioStream = utils.NewAudioStream(utils.AVMediaTypeAudio, d.audioIndex, codecId, extraData, config.SampleRate, config.Channels)
			n = len(data)
		} else {
			audioStream = utils.NewAVStream(utils.AVMediaTypeAudio, d.audioIndex, codecId, nil, nil)
		}

		d.audioStream = audioStream
		d.Handler.OnDeMuxStream(d.audioStream)
		if d.videoIndex != -1 {
			d.Handler.OnDeMuxStreamDone()
		}

		return nil
	}

	if d.audioStream == nil {
		return fmt.Errorf("missing audio sequence header")
	}

	var duration int64
	if d.audioTs != -1 {
		duration = int64(ts) - d.audioTs
	}

	// 时间戳溢出
	if d.audioTs != -1 && int64(ts) < d.audioTs {
		duration = 0xFFFFFFFF - d.audioTs + int64(ts)
	}

	// 根据采样率计算出帧长
	if duration == 0 {
		if utils.AVCodecIdAAC == d.audioStream.CodecId() {
			duration = int64(utils.ComputeAACFrameDuration(d.audioStream.(*utils.AudioStream).SampleRate))
		} else if utils.AVCodecIdPCMALAW == d.audioStream.CodecId() || utils.AVCodecIdPCMMULAW == d.audioStream.CodecId() {
			duration = int64(len(data[n:]))
		}
	}

	d.audioTs = int64(ts)
	packet := utils.NewAudioPacket(data[n:], d.audioTs, d.audioTs, codecId, d.audioIndex, 1000)
	packet.SetDuration(duration)
	d.Handler.OnDeMuxPacket(packet)
	return nil
}

// ParseAudioData 解析音频数据
// @return int 音频帧起始偏移量，例如AAC AUDIO DATA跳过pkt type后的位置
// @return bool 是否是sequence header
func ParseAudioData(data []byte) (int, bool, utils.AVCodecID, error) {
	if len(data) < 4 {
		return -1, false, utils.AVCodecIdNONE, fmt.Errorf("invalid data")
	}

	soundFormat := data[0] >> 4
	//aac
	if byte(SoundFormatAAC) == soundFormat {
		//audio sequence header
		if data[1] == 0x0 {
			/*if len(data) < 4 {
				return -1, false, SoundFormat(0), fmt.Errorf("MPEG4 Audio Config requires at least 2 bytes")
			}*/

			return 2, true, utils.AVCodecIdAAC, nil
		} else if data[1] == 0x1 {
			return 2, false, utils.AVCodecIdAAC, nil
		}
	} else if byte(SoundFormatMP3) == soundFormat {
		return 1, false, utils.AVCodecIdMP3, nil
	} else if byte(SoundFormatG711A) == soundFormat {
		return 1, false, utils.AVCodecIdPCMALAW, nil
	} else if byte(SoundFormatG711B) == soundFormat {
		return 1, false, utils.AVCodecIdPCMMULAW, nil
	} else if byte(SoundFormatExHeader) == soundFormat {

	}

	return -1, false, utils.AVCodecIdNONE, fmt.Errorf("the codec %d is currently not supported in FLV", soundFormat)
}

// ParseVideoData 解析视频数据
// @return int 本次解析了多长字节数
// @return bool 是否是SequenceHeader
// @return bool 是否是关键帧
// @return utils.AVCodecID 视频编码ID
// @return int CompositionTime
func ParseVideoData(data []byte) (int, bool, bool, utils.AVCodecID, int, error) {
	if len(data) < 6 {
		return -1, false, false, utils.AVCodecIdNONE, 0, fmt.Errorf("invaild data")
	}

	frameType := data[0] >> 4
	codeId := data[0] & 0xF

	if frameType == 5 {
		return 0, false, false, utils.AVCodecIdNONE, 0, nil
	}

	if byte(VideoCodeIdH264) == codeId {
		pktType := data[1]
		ct := libbufio.Uint24(data[2:])

		return 5, pktType == 0, frameType == 1, utils.AVCodecIdH264, int(ct), nil
	} else if byte(VideoCodeIdH263) == codeId {
		//pktType := data[1]
		//ct := utils.Uint24(data[2], data[3], data[4])
		pktType := 1
		ct := 0
		return 0, pktType == 0, frameType == 1, utils.AVCodecIdH263, int(ct), nil
	} else if int(frameType)&0b1000 != 0 {
		pktType := PacketType(codeId)
		if PacketTypeMetaData != pktType {
			frameType &= 0x7
		}

		fourCC := binary.BigEndian.Uint32(data[1:])
		var ct int
		n := 5

		if uint32(VideoCodeIdAV1) == fourCC {

		} else if uint32(VideoCodeIdVP9) == fourCC {

		} else if uint32(VideoCodeIdHEVC) == fourCC {
			if PacketTypeSequenceStart == pktType {
			} else if PacketTypeCodedFrames == pktType || PacketTypeCodedFramesX == pktType {
				if PacketTypeCodedFrames == pktType {
					ct = int(libbufio.Uint24(data[5:]))
					n += 3
				}
			} else if PacketTypeMetaData == pktType {

				//if _, err := DoReadAMF0(data[5:]); err != nil {
				//	return 0, false, false, 0, 0, err
				//}

				return 0, false, false, utils.AVCodecIdNONE, 0, nil
			} else if PacketTypeSequenceEnd == pktType {

			}
		} else {
			return -1, false, false, utils.AVCodecIdNONE, 0, fmt.Errorf("unknow codec:%s", string(data[1:5]))
		}

		return n, pktType == 0, frameType == 1, utils.AVCodecIdH265, ct, nil
	}

	return -1, false, false, utils.AVCodecIdNONE, 0, fmt.Errorf("the codec %d is currently not supported in FLV", codeId)
}

func makeFourCc(str string) uint32 {
	utils.Assert(len(str) == 4)
	return binary.BigEndian.Uint32([]byte(str))
}

func NewDeMuxer() DeMuxer {
	return &deMuxer{audioIndex: -1, videoIndex: -1, audioTs: -1, videoTs: -1}
}
