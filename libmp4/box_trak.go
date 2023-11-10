package libmp4

import "github.com/yangjiechina/avformat/utils"

/*
*
Box	Type: ‘tkhd’
Container: Track Box (‘trak’)
Mandatory: Yes
Quantity: Exactly one
*/
type trackHeaderBox struct {
	fullBox
	finalBox
	creationTime     uint64
	modificationTime uint64
	trackId          uint32
	//reserved         uint32
	duration uint64

	//const unsigned int(32)[2] reserved = 0;
	layer          int16
	alternateGroup int16
	volume         int16
	//	const unsigned int(16) reserved = 0;
	matrix [9]int32
	width  int32
	height int32
}

/*
*
Box Type: `tref’
Container: Track Box (‘trak’)
Mandatory: No
Quantity: Zero or one
*/
type trackReferenceBox struct {
	containerBox
}

// ‘hint’
// ‘cdsc‘
// ‘font‘
// ‘hind‘
// ‘vdep’
// ‘vplx’
// ‘subt’
type trackReferenceTypeBox struct {
	finalBox
	referenceType uint32
	trackIds      []uint32
}

/*
*
Box	Type:	 ‘trgr’
Container:	 Track	Box	(‘trak’)
Mandatory:	 No
Quantity:	 Zero	or	one
*/
type trackGroupBox struct {
	containerBox
}

// msrc
type trackGroupTypeBox struct {
	finalBox
	trackGroupType uint32
	fullBox
	trackGroupId uint32
}

/*
*
Box	Type:	 ‘edts’
Container:	 Track	Box	(‘trak’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type editBox struct {
	containerBox
}

/*
*
Box	Type:	 ‘elst’
Container:	 Edit	Box	(‘edts’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type editListBox struct {
	fullBox
	finalBox
	entryCount        uint32
	segmentDuration   []uint64
	mediaTime         []int64
	mediaRateInteger  []int16
	mediaRateFraction []int16
}

/*
*
Box	Type:	 ‘mdia’
Container:	 Track	Box	(‘trak’)
Mandatory:	Yes
Quantity:	 Exactly	one*
*/
type mediaBox struct {
	containerBox
}

func parseTrackHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	tkhd := trackHeaderBox{fullBox: fullBox{version: version, flags: flags}}
	if version == 1 {
		tkhd.creationTime = buffer.ReadUInt64()
		tkhd.modificationTime = buffer.ReadUInt64()
		tkhd.trackId = buffer.ReadUInt32()
		buffer.Skip(4)
		tkhd.duration = buffer.ReadUInt64()
	} else { // version==0
		tkhd.creationTime = uint64(buffer.ReadUInt32())
		tkhd.modificationTime = uint64(buffer.ReadUInt32())
		tkhd.trackId = buffer.ReadUInt32()
		buffer.Skip(4)
		tkhd.duration = uint64(buffer.ReadUInt32())
	}

	buffer.Skip(8)
	tkhd.layer = buffer.ReadInt16()
	tkhd.alternateGroup = buffer.ReadInt16()
	tkhd.volume = buffer.ReadInt16()
	//	const unsigned int(16) reserved = 0;
	buffer.Skip(2)
	//matrix [9]int32
	buffer.Skip(36)
	tkhd.width = buffer.ReadInt32()
	tkhd.height = buffer.ReadInt32()

	return &tkhd, len(data), nil
}

func parseTrackReferenceBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &trackReferenceBox{}, containersBoxConsumeCount, nil
}

func parseTrackReferenceTypeBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	trefType := trackReferenceTypeBox{}
	trefType.referenceType = buffer.ReadUInt32()

	for i := 4; i < len(data) && len(data)%4 == 0; i++ {
		trefType.trackIds = append(trefType.trackIds, buffer.ReadUInt32())
		i += 4
	}

	return &trefType, len(data), nil
}

func parseTrackGroupBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &trackGroupBox{}, containersBoxConsumeCount, nil
}

func parseTrackGroupTypeBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	trgr := trackGroupTypeBox{fullBox: fullBox{version: version, flags: flags}}
	trgr.trackGroupType = buffer.ReadUInt32()
	trgr.trackGroupId = buffer.ReadUInt32()
	return &trgr, containersBoxConsumeCount, nil
}

func parseEditBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &editBox{}, containersBoxConsumeCount, nil
}

func parseEditListBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	elst := editListBox{fullBox: fullBox{version: version, flags: flags}}
	elst.entryCount = buffer.ReadUInt32()
	for i := 0; i < int(elst.entryCount); i++ {
		if elst.version == 1 {
			elst.segmentDuration = append(elst.segmentDuration, buffer.ReadUInt64())
			elst.mediaTime = append(elst.mediaTime, buffer.ReadInt64())
		} else { // version==0
			elst.segmentDuration = append(elst.segmentDuration, uint64(buffer.ReadUInt32()))
			elst.mediaTime = append(elst.mediaTime, int64(buffer.ReadInt32()))
		}
		elst.mediaRateInteger = append(elst.mediaRateInteger, buffer.ReadInt16())
		elst.mediaRateFraction = append(elst.mediaRateFraction, buffer.ReadInt16())
	}

	ctx.tracks[len(ctx.tracks)-1].mark |= markEditLit
	ctx.tracks[len(ctx.tracks)-1].elst = &elst
	return &elst, len(data), nil
}

func parseMediaBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &mediaBox{}, containersBoxConsumeCount, nil
}
