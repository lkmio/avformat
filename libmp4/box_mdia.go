package libmp4

import (
	"fmt"
	"github.com/yangjiechina/avformat/utils"
)

/*
*
Box	Type:	 ‘mdhd’
Container:	 Media	Box	(‘mdia’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type mediaHeaderBox struct {
	fullBox
	containerBox
	creationTime     uint64
	modificationTime uint64
	timescale        uint32
	duration         uint64
	pad              bool
	language         [3]byte
	preDefined       uint16
}

/*
*
Box	Type:	 ‘hdlr’
Container:	 Media	Box	(‘mdia’)	or	Meta	Box	(‘meta’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type handlerReferenceBox struct {
	fullBox
	finalBox

	preDefined  uint32
	handlerType uint32
	//const unsigned int(32)[3] reserved = 0
	name string
}

/*
*
Box	Type:	 ‘elng’
Container:	 Media	Box	(‘mdia’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type extendedLanguageBox struct {
	fullBox
	finalBox
	extendedLanguage string
}

/*
*
Box	Type:	 ‘minf’
Container:	 Media	Box	(‘mdia’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type mediaInformationBox struct {
	containerBox
}

/*
*
Box	Type:	 ‘stbl’
Container:	 Media	Information	Box	(‘minf’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type sampleTableBox struct {
	containerBox
}

func parseMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	mdhd := mediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	if version == 1 {
		mdhd.creationTime = buffer.ReadUInt64()
		mdhd.modificationTime = buffer.ReadUInt64()
		mdhd.timescale = buffer.ReadUInt32()
		mdhd.duration = buffer.ReadUInt64()
	} else { // version==0
		mdhd.creationTime = uint64(buffer.ReadUInt32())
		mdhd.modificationTime = uint64(buffer.ReadUInt32())
		mdhd.timescale = buffer.ReadUInt32()
		mdhd.duration = uint64(buffer.ReadUInt32())
	}

	language := buffer.ReadUInt16()
	mdhd.pad = language>>15 == 1
	mdhd.language[0] = byte(language >> 10 & 0x1F)
	mdhd.language[1] = byte(language >> 5 & 0x1F)
	mdhd.language[2] = byte(language & 0x1F)
	mdhd.preDefined = buffer.ReadUInt16()

	ctx.tracks[len(ctx.tracks)-1].mark |= markMediaHeader
	ctx.tracks[len(ctx.tracks)-1].mdhd = &mdhd
	return &mdhd, len(data), nil
}

func parseHandlerReferenceBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	hdlr := handlerReferenceBox{fullBox: fullBox{version: version, flags: flags}}
	hdlr.preDefined = buffer.ReadUInt32()
	hdlr.handlerType = buffer.ReadUInt32()
	buffer.Skip(12)
	hdlr.name = string(data[24:])

	return &hdlr, len(data), nil
}

func parseExtendedLanguageBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	elng := extendedLanguageBox{fullBox: fullBox{version: version, flags: flags}}
	elng.extendedLanguage = string(data[4:])
	return &elng, len(data), nil
}

func parseMediaInformationBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	size := buffer.ReadUInt32()
	buffer.ReadUInt32()
	name := string(data[4:8])

	var metaData MetaData
	switch name {
	case mediaHandlerTypeVideo:
		metaData = &VideoMetaData{}
		metaData.setMediaType(utils.AVMediaTypeVideo)
		break
	case mediaHandlerTypeAudio:
		metaData = &AudioMetaData{}
		metaData.setMediaType(utils.AVMediaTypeAudio)
		break
	case mediaHandlerTypeHint:
		return nil, -1, fmt.Errorf("not processed for %s box", name)
	case mediaHandlerTypeSubTitle:
		metaData = &SubTitleMetaData{}
		metaData.setMediaType(utils.AVMediaTypeSubtitle)
		break
	case mediaHandlerTypeNull:
		return nil, -1, fmt.Errorf("not processed for %s box", name)
	default:
		return nil, -1, fmt.Errorf("unknow box:%s", name)
	}

	ctx.tracks[len(ctx.tracks)-1].metaData = metaData
	parse := parsers[name]
	if b, _, err := parse(ctx, data[8:size]); err != nil {
		return nil, -1, err
	} else {
		m := &mediaInformationBox{}
		m.addChild(b)
		return m, containersBoxConsumeCount + int(size), nil
	}
}

func parseSampleTableBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &sampleTableBox{}, containersBoxConsumeCount, nil
}
