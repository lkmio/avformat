package libmpeg

import (
	"fmt"
	"github.com/lkmio/avformat/libavc"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
)

type esHandler func(data []byte, total int, first bool, mediaType utils.AVMediaType, id utils.AVCodecID,
	dts int64, pts int64) error

// PSDeMuxer PS流解复用器
type PSDeMuxer struct {
	//内部不做拷贝, 只要解析到es数据就回调出去
	handler esHandler

	packetHeader     *PacketHeader
	systemHeader     *SystemHeader
	programStreamMap *ProgramStreamMap
	pesHeader        *PESHeader
	reader           libbufio.BytesReader

	esCount   uint16 // 已经读取到的ES流数量
	codecId   utils.AVCodecID
	mediaType utils.AVMediaType
}

func (d *PSDeMuxer) Close() {
	// 回调最后一帧
	d.handler = nil
}

func (d *PSDeMuxer) SetHandler(handler esHandler) {
	d.handler = handler
}

// 读取并解析非pes头
// @Return int 0-读取到pes头/-1-需要更多数据
func (d *PSDeMuxer) readHeader(reader libbufio.BytesReader) int {
	for {
		startCode := libavc.FindStartCodeWithReader(reader)
		if startCode < 0 {
			return -1
		}

		// 将读取位置回退4个字节，因为起始码占用了4个字节
		_ = reader.SeekBack(4)
		var n int
		if startCode == 0xBA {
			n = readPackHeader(d.packetHeader, reader.RemainingBytes())
		} else if startCode == 0xBB {
			n = readSystemHeader(d.systemHeader, reader.RemainingBytes())
		} else if startCode == 0xBC {
			n, _ = readProgramStreamMap(d.programStreamMap, reader.RemainingBytes())
		} else if StreamIdPrivateStream1 == startCode || StreamIdPaddingStream == startCode || StreamIdPrivateStream2 == startCode {
			//PrivateStream1解析可参考https://github.com/FFmpeg/FFmpeg/blob/release/7.0/libavformat/mpeg.c#L361
			//PrivateStream2解析可以参考https://github.com/FFmpeg/FFmpeg/blob/release/7.0/libavformat/mpeg.c#L266
			_ = reader.Seek(4)

			skipCount, err := reader.ReadUint16()
			if err != nil {
				//需要更多数据
				_ = reader.SeekBack(4)
				return -1
			} else if reader.Seek(int(skipCount)) != nil {
				//需要更多数据
				_ = reader.SeekBack(6)
				return -1
			}
		} else if !((startCode >= 0xc0 && startCode <= 0xdf) || (startCode >= 0xe0 && startCode <= 0xef)) {
			//查找下一个有效数据
			_ = reader.Seek(4)
		} else {
			//找到pes包
			break
		}

		if startCode < 0xBD {
			if n < 1 {
				//需要更多数据
				return -1
			}

			_ = reader.Seek(n)
		}
	}

	return 0
}

func (d *PSDeMuxer) callbackES(data []byte) error {
	length := len(data)
	if length == 0 {
		return nil
	}

	d.esCount += uint16(length)
	completed := d.esCount == d.pesHeader.esLength

	//完善时间戳, 回调要同时包含dts和pts
	dts := d.pesHeader.dts
	pts := d.pesHeader.pts

	//不包含pts使用dts
	//解析pes头时, 已经确保至少包含一个时间戳
	if d.pesHeader.ptsDtsFlags>>1 == 0 {
		pts = dts
	}
	//不包含dts使用dts
	if d.pesHeader.ptsDtsFlags&0x1 == 0 {
		dts = pts
	}

	err := d.handler(data, int(d.pesHeader.esLength), d.esCount == uint16(len(data)), d.mediaType, d.codecId, dts, pts)
	if completed {
		d.esCount = 0
		d.pesHeader.esLength = 0
	}

	return err
}

// Input 确保输入流的连续性, 比如一个视频帧有多个PES包, 多个PES包必须是连续的, 不允许插入非当前帧PES包.
func (d *PSDeMuxer) Input(data []byte) (int, error) {
	d.reader.Reset(data)

	for d.reader.ReadableBytes() > 0 {
		need := d.pesHeader.esLength - d.esCount
		if need > 0 {
			consume := libbufio.MinInt(int(need), d.reader.ReadableBytes())

			bytes, _ := d.reader.ReadBytes(consume)
			if err := d.callbackES(bytes); err != nil {
				return d.reader.Offset(), err
			}
			continue
		}

		n := d.readHeader(d.reader)
		if len(d.programStreamMap.elementaryStreams) < 1 {
			fmt.Printf("skipped %d invalid data bytes\r\n", len(data))
			return len(data), nil
		} else if n < 0 {
			break
		}

		n = readPESHeader(d.pesHeader, d.reader.RemainingBytes())
		if n < 1 {
			break
		}

		_ = d.reader.Seek(n)
		elementaryStream, b := d.programStreamMap.findElementaryStream(d.pesHeader.streamId)
		if !b {
			fmt.Printf("unknow stream id:%x \r\n", d.pesHeader.streamId)
		}

		d.mediaType = utils.AVMediaTypeAudio
		if elementaryStream.streamType == StreamTypeVideoH264 {
			d.codecId = utils.AVCodecIdH264
			d.mediaType = utils.AVMediaTypeVideo
		} else if elementaryStream.streamType == StreamTypeVideoHEVC {
			d.codecId = utils.AVCodecIdH265
			d.mediaType = utils.AVMediaTypeVideo
		} else if elementaryStream.streamType == StreamTypeAudioAAC {
			d.codecId = utils.AVCodecIdAAC
		} else if elementaryStream.streamType == StreamTypeAudioG711A {
			d.codecId = utils.AVCodecIdPCMALAW
		} else if elementaryStream.streamType == StreamTypeAudioG711U {
			d.codecId = utils.AVCodecIdPCMMULAW
		} else {
			return -1, fmt.Errorf("the stream type %d is not implemented", elementaryStream.streamType)
		}
	}

	return d.reader.Offset(), nil
}

func NewPSDeMuxer() *PSDeMuxer {
	return &PSDeMuxer{
		packetHeader:     &PacketHeader{},
		systemHeader:     &SystemHeader{},
		programStreamMap: &ProgramStreamMap{},
		pesHeader:        &PESHeader{},
		reader:           libbufio.NewBytesReader(nil),
	}
}
