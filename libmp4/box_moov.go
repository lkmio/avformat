package libmp4

import (
	"github.com/yangjiechina/avformat/utils"
)

/*
*
8.2.2.2
Box	Type: ‘mvhd’
Container: Movie Box (‘moov’)
Mandatory: Yes
Quantity: Exactly one
*/
type movieHeaderBox struct {
	fullBox
	finalBox
	creationTime     uint64
	modificationTime uint64
	timescale        uint32
	duration         uint64

	rate   uint32 // typically 1.0
	volume uint16 // typically, full volume
	//const bit(16) reserved = 0
	//const unsigned int(32)[2] reserved = 0
	matrix      [9]uint32 //0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000
	preDefined  [6]uint32
	nextTrackId uint32
}

func (m *movieHeaderBox) fixedSize() int {
	if m.version == 1 {
		return 112
	} else {
		return 100
	}
}

/*
Box	Type: ‘trak’
Container: Movie Box (‘moov’)
Mandatory: Yes
Quantity: One or more
*/
type trackBox struct {
	containerBox
}

/*
*
Box	Type:	 ‘udta’
Container:	 Movie	Box	(‘moov’),	Track	Box	(‘trak’),

	Movie	Fragment	Box	(‘moof’)	or	Track	Fragment	Box	(‘traf’)

Mandatory:	No
Quantity:	 Zero	or	one
*/
type userDataBox struct {
	finalBox
}

func parseMovieHeaderBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	mvhd := movieHeaderBox{fullBox: fullBox{version: version, flags: flags}}

	if version == 1 {
		mvhd.creationTime = buffer.ReadUInt64()
		mvhd.modificationTime = buffer.ReadUInt64()
		mvhd.timescale = buffer.ReadUInt32()
		mvhd.duration = buffer.ReadUInt64()
	} else { // version==0
		mvhd.creationTime = uint64(buffer.ReadUInt32())
		mvhd.modificationTime = uint64(buffer.ReadUInt32())
		mvhd.timescale = buffer.ReadUInt32()
		mvhd.duration = uint64(buffer.ReadUInt32())
	}

	mvhd.rate = buffer.ReadUInt32()
	mvhd.volume = buffer.ReadUInt16()
	buffer.Skip(10) //reserved
	buffer.Skip(36) //matrix
	buffer.Skip(24) //preDefined
	mvhd.nextTrackId = buffer.ReadUInt32()

	return &mvhd, len(data), nil
}

func parseTrackBox(ctx *deMuxContext, data []byte) (box, int, error) {
	ctx.tracks = append(ctx.tracks, &Track{})
	return &trackBox{}, containersBoxConsumeCount, nil
}

func parseUserDataBox(ctx *deMuxContext, data []byte) (box, int, error) {
	return &userDataBox{}, len(data), nil
}
