package libmpeg

import (
	"bufio"
	"fmt"
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/utils"
	"io"
	"io/ioutil"
	"os"
)

type deHandler func(buffer utils.ByteBuffer, keyFrame bool, streamType int, pts, dts int64)

type DeMuxer struct {
	handler          deHandler
	packetHeader     *PacketHeader
	systemHeader     *SystemHeader
	programStreamMap *ProgramStreamMap
	lastPesHeader    *PESHeader
	currentPesHeader *PESHeader

	packet     *utils.AVPacket
	streamType byte
}

func callbackES(streamId, streamType byte, packet *utils.AVPacket, handler deHandler) {
	var keyFrame bool
	switch streamId {
	case StreamIdAudio:
		keyFrame = true
		break
	case StreamIdVideo, StreamIdH624:
		keyFrame = libavc.IsKeyFrameFromBuffer(packet.Data())
		break
	}

	handler(packet.Data(), keyFrame, int(streamType), packet.Pts(), packet.Dts())
}

func (d *DeMuxer) Close() {
	//回调最后一帧
	if d.packet.Data().Size() > 0 {
		callbackES(d.lastPesHeader.streamId, d.streamType, d.packet, d.handler)
	}
}

//func (d *DeMuxer) callback(streamId byte) {
//	var keyFrame bool
//	switch streamId {
//	case StreamIdAudio:
//		keyFrame = true
//		break
//	case StreamIdVideo, StreamIdH624:
//		keyFrame = libavc.IsKeyFrameFromBuffer(d.packet.Data())
//		break
//	}
//
//	d.handler(d.packet.Data(), keyFrame, int(d.streamType), d.packet.Pts(), d.packet.Dts())
//}

// Input Reference from https://github.com/ireader/media-server/blob/master/libmpeg/source/mpeg-ps-dec.c
func (d *DeMuxer) Input(data []byte) int {
	n, i, consume := 0, 0, 0
	//保存第一个pes的开始位置
	//每次Input如果没有读取到完整的一帧，回退到第一个pes的位置
	//内部不做内存拷贝，ByteBuffer只是浅拷贝
	var firstPesIndex int
	length := len(data)
	d.packet.Release()

	for i = libavc.FindStartCode(data, 0); i >= 0 && i < length; i = libavc.FindStartCode(data, i) {
		i -= 3
		switch data[i+3] {
		case 0xBA:
			n = readPackHeader(d.packetHeader, data[i:])
			break
		case 0xBB:
			n = readSystemHeader(d.systemHeader, data[i:])
			break
		case 0xBC:
			n, _ = readProgramStreamMap(d.programStreamMap, data[i:])
			break
		case 0xB9: //end code
			break
		default:
			var esPacket []byte
			if firstPesIndex == 0 {
				firstPesIndex = i
			}

			n = readPESHeader(d.currentPesHeader, data[i:])
			if n == 0 || len(data[i:])-n > int(d.currentPesHeader.packetLength-3-uint16(d.currentPesHeader.pesHeaderDataLength)) {
				goto END
			}

			element, ok := d.programStreamMap.findElementaryStream(data[i+3])
			if !ok {
				println(fmt.Sprintf("unknow stream:%x", data[i+3]))
				break
			}

			if d.lastPesHeader == nil {
				pesPacket := *d.currentPesHeader
				d.lastPesHeader = &pesPacket
			}

			//读到下一包，才回调前一包
			//上一包和当前包的pts/streamId不一样,才回调
			if d.currentPesHeader.streamId != d.lastPesHeader.streamId || d.currentPesHeader.pts != d.lastPesHeader.pts {
				//d.callback(d.lastPesHeader.streamId)
				callbackES(d.lastPesHeader.streamId, d.streamType, d.packet, d.handler)
				d.packet.Release()
				*d.lastPesHeader = *d.currentPesHeader
				firstPesIndex = i
			}

			d.streamType = element.streamType
			if d.currentPesHeader.ptsDtsFlags&0x3 != 0 {
				d.packet.SetPts(d.currentPesHeader.pts)
				d.packet.SetDts(d.currentPesHeader.dts)
			}
			d.packet.Write(esPacket)
			d.currentPesHeader.Reset()
		}

		i += n
		consume = i
	}

END:
	d.currentPesHeader.Reset()
	if firstPesIndex != 0 {
		return firstPesIndex
	} else {
		return consume
	}
}

// Open 解复用本地文件
// @readCount 每次读取多少字节. <= 0 一次性读取完
func (d *DeMuxer) Open(path string, readCount int) error {
	if readCount <= 0 {
		all, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		d.Input(all)
		return nil
	} else {
		fi, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			fi.Close()
		}()

		reader := bufio.NewReader(fi)
		offset := 0
		buffer := make([]byte, readCount)

		for {
			r, err := reader.Read(buffer[offset:])
			if err != nil {
				if err == io.EOF {
					return nil
				} else {
					return err
				}
			}

			length := offset + r
			consume := d.Input(buffer[:length])
			offset = length - consume
			copy(buffer, buffer[consume:length])
		}
	}
}

func NewDeMuxer(handler deHandler) *DeMuxer {
	return &DeMuxer{
		handler:          handler,
		packetHeader:     &PacketHeader{},
		systemHeader:     &SystemHeader{},
		programStreamMap: &ProgramStreamMap{},
		currentPesHeader: NewPESPacket(),
		packet:           utils.NewPacket(),
	}
}
