package libmp4

import "github.com/yangjiechina/avformat/utils"

/*
*
Box	Types:	 ‘vmhd’
Container:	 Media	Information	Box	(‘minf’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type videoMediaHeaderBox struct {
	fullBox
	finalBox
	graphicsMode uint16 // copy, see below
	opColor      [3]uint16
}

/*
*
Box	Types:	 	‘smhd’
Container:	 Media	Information	Box	(‘minf’)
Mandatory:	Yes
Quantity:	 Exactly	one	specific	media	header	shall	be	present
*/
type soundMediaHeaderBox struct {
	fullBox
	finalBox
	balance int16
	//const unsigned int(16) reserved = 0;
}

/*
*
Box	Types:	 ’hmhd’
Container:	 Media	Information	Box	(‘minf’)
Mandatory:	Yes
Quantity:	 Exactly	one	specific	media	header	shall	be	present
*/
type hintMediaHeaderBox struct {
	fullBox
	finalBox
	maxPDUSize uint16
	avgPDUSize uint16
	maxBitrate uint32
	avgBitrate uint32
	//reserved uint32
}

/*
*
‘sthd’
*/
type subtitleMediaHeaderBox struct {
	fullBox
	finalBox
}

/*
*
nmhd
*/
type nullMediaHeaderBox struct {
	fullBox
	finalBox
}

/*
*
Box	Type:	 ‘dinf’
Container:	 Media	Information	Box	(‘minf’)	or	Meta	Box	(‘meta’)
Mandatory:	Yes	(required	within	‘minf’	box)	and	No	(optional	within	‘meta’	box)
Quantity:	 Exactly	one
*/
type dataInformationBox struct {
	containerBox
}

/*
*
Box	Types:	‘dref’
Container:	Data	Information	Box	(‘dinf’)
Mandatory:	Yes
Quantity:	Exactly	one
*/
type dataReferenceBox struct {
	containerBox
	entryCount uint32
}

/*
*
Box	Types:	‘url ‘,	‘urn ‘
Container:	Data	Information	Box	(‘dref’)
Mandatory:	Yes	(at	least	one	of	‘url	‘	or	‘urn	‘	shall	be	present)
Quantity:	One	or	more
*/
type dataEntryUrlBox struct {
	fullBox
	finalBox
	location string
}

type dataEntryUrnBox struct {
	fullBox
	finalBox
	name     string
	location string
}

func parseVideoMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	vmhd := videoMediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	vmhd.graphicsMode = buffer.ReadUInt16()
	vmhd.opColor[0] = buffer.ReadUInt16()
	vmhd.opColor[1] = buffer.ReadUInt16()
	vmhd.opColor[2] = buffer.ReadUInt16()
	return &vmhd, len(data), nil
}

func parseSoundMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	smhd := soundMediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	smhd.balance = buffer.ReadInt16()
	return &smhd, len(data), nil
}

func parseHintMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	hmhd := hintMediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	hmhd.maxPDUSize = buffer.ReadUInt16()
	hmhd.avgPDUSize = buffer.ReadUInt16()
	hmhd.maxBitrate = buffer.ReadUInt32()
	hmhd.avgBitrate = buffer.ReadUInt32()
	return &hmhd, len(data), nil
}

func parseSubtitleMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	sthd := subtitleMediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	return &sthd, len(data), nil
}

func parseNullMediaHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	nmhd := nullMediaHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	return &nmhd, len(data), nil
}

func parseDataInformationBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &dataInformationBox{}, containersBoxConsumeCount, nil
}

func parseDataReferenceBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &dataReferenceBox{}, containersBoxConsumeCount, nil
}

func parseDataEntryUrlBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	url := dataEntryUrlBox{fullBox: fullBox{version: version, flags: flags}}
	url.location = string(data[4:])
	return &url, len(data), nil
}

func parseDataEntryUrnBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	url := dataEntryUrnBox{fullBox: fullBox{version: version, flags: flags}}
	url.location = string(data[4:])
	return &url, len(data), nil
}
