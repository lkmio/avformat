package libflv

import (
	"encoding/binary"
	"fmt"
	"github.com/yangjiechina/avformat"
	"github.com/yangjiechina/avformat/utils"
)

type TagType byte
type VideoCodecId byte
type SoundFormat byte
type SoundRate byte

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

	SoundFormatMP3   = SoundFormat(2)
	SoundFormatG711A = SoundFormat(7)
	SoundFormatG711B = SoundFormat(8)
	SoundFormatAAC   = SoundFormat(10)
	SoundFormatMP38K = SoundFormat(14)

	SoundRate5500HZ  = SoundRate(0)
	SoundRate11000HZ = SoundRate(1)
	SoundRate22000HZ = SoundRate(2)
	SoundRate44000HZ = SoundRate(3) //For AAC:always 3
)

type DeMuxer struct {
	avformat.DeMuxerImpl

	/**
	duration: DOUBLE
	width: DOUBLE
	height: DOUBLE
	videodatarate: DOUBLE
	framerate: DOUBLE
	videocodecid: DOUBLE
	audiosamplerate: DOUBLE
	audiosamplesize: DOUBLE
	stereo: BOOL
	audiocodecid: DOUBLE
	filesize: DOUBLE
	*/
	metaData []interface{}

	headerCompleted bool
	//保存当前正在读取的Tag
	tag Tag
}

type Tag struct {
	preSize   uint32
	type_     TagType
	dataSize  int
	timestamp uint32

	data []byte
	size int
}

func (d *DeMuxer) readScriptDataObject(data []byte) error {
	buffer := utils.NewByteBuffer(data)

	if err := buffer.PeekCount(1); err != nil {
		return err
	}

	metaData, err := DoReadAFM0FromBuffer(buffer)
	if err != nil {
		return err
	}
	if len(metaData) <= 0 {
		return fmt.Errorf("invalid data")
	}
	if s, ok := metaData[0].(string); s == "" || !ok {
		return fmt.Errorf("not find the ONMETADATA of AMF0")
	}

	d.metaData = metaData
	return nil
}

/*
func Valid(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return false, err
	}

	h := make([]byte, 9)
	n, err := file.Read(h)

	if err != nil {
		return false, err
	}
	if n < 9 {
		return false, fmt.Errorf("flv探测数据不够 size:%d", n)
	}

	if n < 9 {
		return false, fmt.Errorf("flv探测数据不够 size:%d", n)
	}

	if h[0] != 0x46 || h[1] != 0x4C || h[2] != 0x56 {
		return false, fmt.Errorf("invalid data")
	}

	version := h[3]
	flags := typeFlag(h[4])
	dataOffset := binary.BigEndian.Uint32(h[5:])
	if version == 1 && dataOffset != 9 {
		return false, fmt.Errorf("invalid data")
	}

	if !flags.ExistAudio() && !flags.ExistAudio() {
		return false, fmt.Errorf("invalid data")
	}

	tagHeader := make([]byte, 15)
	var offset int64
	offset = 9

	for {
		n, err = file.ReadAt(tagHeader, offset)
		if err != nil {
			return false, err
		}
		if n < 15 {
			return false, fmt.Errorf("flv探测数据不够 size:%d", n)
		}

		//pre size
		_ = binary.BigEndian.Uint32(tagHeader)
		tagType := tagHeader[4]
		dataSize := binary.BigEndian.Uint32(tagHeader[5:]) >> 8

		if TagTypeAudioData == TagType(tagType) || TagTypeVideoData == TagType(tagType) {
			return true, nil
		}

		offset += 15 + int64(dataSize)
		if stat.Size() < offset {
			return false, fmt.Errorf("invalid data")
		}
	}
}*/

func (d *DeMuxer) readHeader(data []byte) error {
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
func (d *DeMuxer) readTag(data []byte) Tag {
	_ = data[15]
	timestamp := utils.BytesToUInt24WhitSlice(data[8:])
	timestamp |= uint32(data[11]) << 24

	return Tag{preSize: binary.BigEndian.Uint32(data), type_: TagType(data[4]), dataSize: int(utils.BytesToUInt24WhitSlice(data[5:])), timestamp: timestamp}
}

func (d *DeMuxer) parseTag(data []byte, tagType TagType, ts uint32) error {
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

func (d *DeMuxer) Input(data []byte) (int, error) {
	var n int
	if !d.headerCompleted {
		if err := d.readHeader(data); err != nil {
			return -1, err
		}

		d.headerCompleted = true
		n = 9
	}

	//读取未解析完的Tag
	need := d.tag.dataSize - d.tag.size
	if need > 0 {
		min := utils.MinInt(len(data), need)
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

	for len(data[n:]) > 15 {
		tag := d.readTag(data[n:])
		n += 15

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

func (d *DeMuxer) InputVideo(data []byte, ts uint32) error {
	n, sequenceHeader, key, id, ct, err := ParseVideoData(data)
	if err != nil {
		return err
	}

	if sequenceHeader {
		var stream utils.AVStream
		var codecId utils.AVCodecID
		if id == VideoCodeIdH264 {
			codecId = utils.AVCodecIdH264

		}

		stream = utils.NewAVStream(utils.AVMediaTypeVideo, 0, codecId, data[n:], utils.ExtraTypeM4VC)
		d.Handler.OnDeMuxStream(stream)
	} else {
		var packet *utils.AVPacket2
		var codecId utils.AVCodecID
		if id == VideoCodeIdH264 {
			codecId = utils.AVCodecIdH264
		}

		packet = utils.NewAVPacket2(data[n:], int64(ts), int64(ts+uint32(ct)), key, utils.PacketTypeAVCC, utils.AVMediaTypeVideo, codecId)
		d.Handler.OnDeMuxPacket(0, packet)
	}

	return nil
}

func (d *DeMuxer) InputAudio(data []byte, ts uint32) error {
	n, sequenceHeader, format, err := ParseAudioData(data)
	if err != nil {
		return err
	}

	if sequenceHeader {
		var stream utils.AVStream
		var codecId utils.AVCodecID
		if format == SoundFormatAAC {
			codecId = utils.AVCodecIdAAC
		}

		stream = utils.NewAVStream(utils.AVMediaTypeAudio, 0, codecId, data[n:], utils.ExtraTypeNONE)
		d.Handler.OnDeMuxStream(stream)
	} else {
		var packet *utils.AVPacket2
		var codecId utils.AVCodecID
		if format == SoundFormatAAC {
			codecId = utils.AVCodecIdAAC
		}

		packet = utils.NewAVPacket2(data[n:], int64(ts), int64(ts), true, utils.PacketTypeNONE, utils.AVMediaTypeAudio, codecId)
		d.Handler.OnDeMuxPacket(0, packet)
	}

	return nil
}

// ParseAudioData 解析音频数据
// int 音频帧起始偏移量，例如AAC AUDIO DATA跳过pkt type后的位置
// bool 是否是sequence header
func ParseAudioData(data []byte) (int, bool, SoundFormat, error) {
	if len(data) < 4 {
		return -1, false, SoundFormat(0), fmt.Errorf("invalid data")
	}

	soundFormat := data[0] >> 4
	//aac
	if soundFormat == 10 {
		//audio sequence header
		if data[1] == 0x0 {
			/*if len(data) < 4 {
				return -1, false, SoundFormat(0), fmt.Errorf("MPEG4 Audio Config requires at least 2 bytes")
			}*/

			return 2, true, SoundFormatAAC, nil
		} else if data[1] == 0x1 {
			return 2, false, SoundFormatAAC, nil
		}
	}

	return -1, false, SoundFormat(0), fmt.Errorf("the codec %d is currently not supported in FLV", soundFormat)
}

func ParseVideoData(data []byte) (int, bool, bool, VideoCodecId, int, error) {
	if len(data) < 6 {
		return -1, false, false, VideoCodecId(0), 0, fmt.Errorf("invaild data")
	}

	frameType := data[0] >> 4
	codeId := data[0] & 0xF

	if byte(VideoCodeIdH264) == codeId {
		pktType := data[1]
		ct := utils.BytesToUInt24(data[2], data[3], data[4])

		return 5, pktType == 0, frameType == 1, VideoCodeIdH264, int(ct), nil
	}

	return -1, false, false, VideoCodecId(0), 0, fmt.Errorf("the codec %d is currently not supported in FLV", codeId)
}
