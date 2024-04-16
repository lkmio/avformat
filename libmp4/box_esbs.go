package libmp4

import (
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
)

const (
	mp4ODescTag           = 0x01
	mp4IODescTag          = 0x02
	mp4ESDescTag          = 0x03
	mp4DecConfigDescTag   = 0x04
	mp4DecSpecificDescTag = 0x05
	mp4SLDescTag          = 0x06
)

var (
	mp4ObjType = map[int]utils.AVCodecID{
		0x08: utils.AVCodecIdMOVTEXT,
		0x20: utils.AVCodecIdMPEG4,
		0x21: utils.AVCodecIdH264,
		0x23: utils.AVCodecIdHEVC,
		0x40: utils.AVCodecIdAAC,
		//0x40: utils.AVCodecIdMP4ALS,     /* 14496-3 ALS */
		0x61: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 Main */
		0x60: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 Simple */
		0x62: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 SNR */
		0x63: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 Spatial */
		0x64: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 High */
		0x65: utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 422 */
		0x66: utils.AVCodecIdAAC,        /* MPEG-2 AAC Main */
		0x67: utils.AVCodecIdAAC,        /* MPEG-2 AAC Low */
		0x68: utils.AVCodecIdAAC,        /* MPEG-2 AAC SSR */
		0x69: utils.AVCodecIdMP3,        /* 13818-3 */
		//0x69: utils.AVCodecIdMP2,        /* 11172-3 */
		0x6A: utils.AVCodecIdMPEG1VIDEO, /* 11172-2 */
		0x6B: utils.AVCodecIdMP3,        /* 11172-3 */
		0x6C: utils.AVCodecIdMJPEG,      /* 10918-1 */
		0x6D: utils.AVCodecIdPNG,
		0x6E: utils.AVCodecIdJPEG2000, /* 15444-1 */
		0xA3: utils.AVCodecIdVC1,
		0xA4: utils.AVCodecIdDIRAC,
		0xA5: utils.AVCodecIdAC3,
		0xA6: utils.AVCodecIdEAC3,
		0xA9: utils.AVCodecIdDTS,         /* mp4ra.org */
		0xAD: utils.AVCodecIdOPUS,        /* mp4ra.org */
		0xB1: utils.AVCodecIdVP9,         /* mp4ra.org */
		0xC1: utils.AVCodecIdFLAC,        /* nonstandard, update when there is a standard value */
		0xD0: utils.AVCodecIdTSCC2,       /* nonstandard, camtasia uses it */
		0xD1: utils.AVCodecIdEVRC,        /* nonstandard, pvAuthor uses it */
		0xDD: utils.AVCodecIdVORBIS,      /* nonstandard, gpac uses it */
		0xE0: utils.AVCodecIdDVDSUBTITLE, /* nonstandard, see unsupported-embedded-subs-2.mp4 */
		0xE1: utils.AVCodecIdQCELP,
		0x01: utils.AVCodecIdMPEG4SYSTEMS,
		0x02: utils.AVCodecIdMPEG4SYSTEMS,
		0:    utils.AVCodecIdNONE,
	}
)

type esdBox struct {
	fullBox
	finalBox
}

func readDesc(buffer libbufio.ByteBuffer) (int, int) {
	tag := buffer.ReadUInt8()
	return int(tag), readDescLen(buffer)
}

func readDescLen(buffer libbufio.ByteBuffer) int {
	length, count := 0, 4
	for ; count > 0; count-- {
		c := buffer.ReadUInt8()
		length = (length << 7) | int(c&0x7f)
		if c&0x80 == 0 {
			break
		}
	}

	return length
}

func parseESDesc(buffer libbufio.ByteBuffer, esId int) {
	var flags int
	if esId != 0 {
		esId = int(buffer.ReadUInt16())
	} else {
		buffer.ReadUInt16()
	}
	flags = int(buffer.ReadUInt8())
	//streamDependenceFlag
	if flags&0x80 != 0 {
		buffer.ReadUInt16()
	}
	//URL_Flag
	if flags&0x40 != 0 {
		length := buffer.ReadUInt8()
		buffer.Skip(int(length))
	}
	//OCRstreamFlag
	if flags&0x20 != 0 {
		buffer.ReadUInt16()
	}
}

func readDecConfigDesc(t *Track, buffer libbufio.ByteBuffer) {
	objectTypeId := buffer.ReadUInt8()
	buffer.ReadUInt8()
	buffer.ReadUInt24()
	//v
	_ = buffer.ReadUInt32()

	//bitRate
	_ = buffer.ReadUInt32()
	tag, length := readDesc(buffer)
	if tag == mp4DecSpecificDescTag {
		if objectTypeId == 0x69 || objectTypeId == 0x6b {
			return
		}
		if length == 0 || length > (1<<30) {
			//invalid data
			return
		}
		bytes := buffer.ReadableBytes()
		if bytes < length {
			return
		}

		extra := make([]byte, bytes)
		buffer.ReadBytes(extra)
		t.metaData.setExtraData(extra)
	}
}

func parseESDBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := libbufio.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	esd := esdBox{fullBox: fullBox{version: version, flags: flags}}
	tag, _ := readDesc(buffer)
	if tag == mp4ESDescTag {
		parseESDesc(buffer, 0)
	} else {
		_ = buffer.ReadUInt16()
	}

	tag, _ = readDesc(buffer)
	if tag == mp4DecConfigDescTag {
		readDecConfigDesc(ctx.tracks[len(ctx.tracks)-1], buffer)
	}
	return &esd, len(data), nil
}
