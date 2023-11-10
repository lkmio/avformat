package libmp4

import (
	"fmt"
	"github.com/yangjiechina/avformat/libavc"
	"github.com/yangjiechina/avformat/libhevc"
	"github.com/yangjiechina/avformat/utils"
)

//stsd * 8.5.2 sample descriptions (codec types, initialization
//etc.)
//stts * 8.6.1.2 (decoding) time-to-sample
//ctts 8.6.1.3 (composition) time to sample
//cslg 8.6.1.4 composition to decode timeline mapping
//stsc * 8.7.4 sample-to-chunk, partial data-offset information
//stsz 8.7.3.2 sample sizes (framing)
//stz2 8.7.3.3 compact sample sizes (framing)
//stco * 8.7.5 chunk offset, partial data-offset information
//co64 8.7.5 64-bit chunk offset
//stss 8.6.2 sync sample table
//stsh 8.6.3 shadow sync sample table
//padb 8.7.6 sample padding bits
//stdp 8.7.6 sample degradation priority
//sdtp 8.6.4 independent and disposable samples
//sbgp 8.9.2 sample-to-group
//sgpd 8.9.3 sample group description
//subs 8.7.7 sub-sample information
//saiz 8.7.8 sample auxiliary information sizes
//saio 8.7.9 sample auxiliary information offsets

/*
*
Box	Types:	 ‘stsd’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type sampleDescriptionBox struct {
	fullBox
	containerBox
	entryCount uint32
}

/*
*
Box	Type:	 ‘stdp’
Container:	 Sample	Table	Box	(‘stbl’).
Mandatory:	No.
Quantity:	 Zero	or	one.
*/
type degradationPriorityBox struct {
	fullBox
	finalBox
	//sampleCount from 'stsz'
	//int i;
	//for (i=0; i < sample_count; i++) {
	//unsigned int(16) priority;
}

/*
*
Box	Type:	 ‘stts’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type decodingTimeToSampleBox struct {
	fullBox
	finalBox
	entryCount  uint32
	sampleCount []uint32
	sampleDelta []uint32
}

/*
*
Box	Type:	 ‘ctts’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type compositionTimeToSampleBox struct {
	fullBox
	finalBox
	entryCount   uint32
	sampleCount  []uint32
	sampleOffset []uint32
}

/*
*
Box	Type:	 ‘cslg’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Extension	Properties	Box	(‘trep’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type compositionToDecodeBox struct {
	fullBox
	finalBox
	compositionToDTSShift        int64
	leastDecodeToDisplayDelta    int64
	greatestDecodeToDisplayDelta int64
	compositionStartTime         int64
	compositionEndTime           int64
}

/*
*
Box	Type:	 ‘stss’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type syncSampleBox struct {
	fullBox
	finalBox
	entryCount uint32
	//sampleNumber []uint32
	sampleNumber map[uint32]byte
}

/*
*
Box	Type:	 ‘stsh’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type shadowSyncSampleBox struct {
	fullBox
	finalBox
	entryCount uint32
}

/*
*
Box	Types:	 ‘sdtp’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type independentAndDisposableSamplesBox struct {
	fullBox
	finalBox
	//sample_count from 'stsz' or ‘stz2’
}

/*
*
Box	Type:	 ‘stsz’,	‘stz2’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	Yes
Quantity:	 Exactly	one	variant	must	be	present
*/
type sampleSizeBox struct {
	fullBox
	finalBox
	sampleSize  uint32
	sampleCount uint32
	entrySize   []uint32
}

// stz2
type compactSampleSizeBox struct {
	fullBox
	finalBox
	fieldSize   uint8 //4/8/16
	sampleCount uint32
	entrySize   []int64
	//	for (i=1; i <= sample_count; i++) {
	//	unsigned int(field_size) entry_size;
	//}
}

/*
*
Box	Type:	 ‘stsc’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	Yes
Quantity:	 Exactly	one
*/
type sampleToChunkBox struct {
	fullBox
	finalBox
	entryCount             uint32
	firstChunk             []uint32
	samplesPerChunk        []uint32 //chunk中sample个数
	sampleDescriptionIndex []uint32
}

/*
*
Box	Type:	 ‘stco’,	‘co64’
Container:	 Sample	Table	Box	(‘stbl’)
Mandatory:	Yes
Quantity:	 Exactly	one	variant	must	be	present
*/
type chunkOffsetBox struct {
	fullBox
	finalBox
	entryCount  uint32
	chunkOffset []uint32
}

// ‘co64’
type chunkLargeOffsetBox struct {
	fullBox
	finalBox
	entryCount  uint32
	chunkOffset []uint64
}

/*
*
Box	Type:	 ‘padb’
Container:	 Sample	Table	(‘stbl’)
Mandatory:	No
Quantity:	 Zero	or	one
*/
type paddingBitsBox struct {
	fullBox
	finalBox
	sampleCount uint32
}

/*
*
Box	Type:	 ‘subs’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Fragment	Box	(‘traf’)
Mandatory:	No
Quantity:	 Zero	or	more
*/
type subSampleInformationBox struct {
	fullBox
	finalBox
	entryCount uint32
}

/*
*
Box	Type:	 ‘saiz’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Fragment	Box	('traf')
Mandatory:	No
Quantity:	 Zero	or	More
*/
type sampleAuxiliaryInformationSizesBox struct {
	fullBox
	finalBox
	auxInfoType           uint32
	auxInfoTypeParameter  uint32
	defaultSampleInfoSize uint8
	sampleCount           uint32
	sampleInfoSize        []uint8
}

/*
*
Box	Type:	 ‘saio’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Fragment	Box	('traf')
Mandatory:	No
Quantity:	 Zero	or	More
*/
type sampleAuxiliaryInformationOffsetsBox struct {
	fullBox
	finalBox
	auxInfoType          uint32
	auxInfoTypeParameter uint32
	entryCount           uint32
	offset               []uint8
}

/*
*
Box	Type:	 ‘sbgp’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Fragment	Box	(‘traf’)
Mandatory:	No
Quantity:	 Zero	or	more
*/
type sampleToGroupBox struct {
	fullBox
	finalBox

	groupingType          uint32
	groupingTypeParameter uint32
	entryCount            uint32
	sampleCount           []uint32
	groupDescriptionIndex []uint32
}

type sampleGroupDescriptionEntry interface {
}

/*
*
Box	Type:	 ‘sgpd’
Container:	 Sample	Table	Box	(‘stbl’)	or	Track	Fragment	Box	(‘traf’)
Mandatory:	No
Quantity:	 Zero	or	more,	with	one	for	each	Sample	to	Group	Box.
*/
type sampleGroupDescriptionBox struct {
	fullBox
	finalBox
	groupingType                  uint32
	defaultLength                 uint32
	defaultSampleDescriptionIndex uint32
	entryCount                    uint32
	descriptionLength             []uint32
	sampleEntries                 []sampleGroupDescriptionEntry
}

func parseSTSDVideo(t *Track, size uint32, buffer utils.ByteBuffer) error {
	offset := buffer.ReadOffset()
	//version
	buffer.ReadUInt16()
	//revision level
	buffer.ReadUInt16()
	//vendor
	_ = buffer.ReadUInt32()
	//temporal quality
	buffer.ReadUInt32()
	//spatial quality
	buffer.ReadUInt32()

	t.metaData.(*VideoMetaData).width = int(buffer.ReadUInt16())
	t.metaData.(*VideoMetaData).height = int(buffer.ReadUInt16())
	//horizSolution fixed value 0x00480000
	buffer.ReadUInt32()
	//vertSolution fixed value 0x00480000
	buffer.ReadUInt32()
	//reserved
	buffer.ReadUInt32()
	//frame count
	_ = buffer.ReadUInt16()
	//compressor name
	buffer.Skip(32)
	//depth fixed value 0x0018
	buffer.ReadUInt16()
	//pre defined -1
	buffer.ReadInt16()

	bytes := buffer.ReadableBytes()
	if bytes > 8 {
		//parse extra
		extraSize := buffer.ReadUInt32() - 8
		buffer.Skip(4)
		if int(extraSize) <= bytes-8 {
			extra := make([]byte, bytes-8)
			buffer.ReadBytes(extra)
			if t.metaData.CodeId() == utils.AVCodecIdH264 {
				spspps := libavc.ExtraDataToAnnexB(extra)
				t.metaData.setExtraData(spspps)
			} else if t.metaData.CodeId() == utils.AVCodecIdHEVC {
				b, l, err := libhevc.ExtraDataToAnnexB(extra)
				if err != nil {
					return err
				}
				t.metaData.(*VideoMetaData).SetLengthSize(l)
				t.metaData.setExtraData(b)
			} else {
				t.metaData.setExtraData(extra)
			}

		}
	}

	consume := buffer.ReadOffset() - offset
	buffer.Skip(int(size) - 16 - consume)
	return nil
}

func parseSTSDAudio(t *Track, size uint32, buffer utils.ByteBuffer) int {
	offset := buffer.ReadOffset()
	//version
	buffer.ReadUInt16()
	//revision level
	buffer.ReadUInt16()
	//vendor
	_ = buffer.ReadUInt32()
	//2
	channelCount := int(buffer.ReadUInt16())
	//16
	sampleSize := int(buffer.ReadUInt16())
	_ = buffer.ReadUInt16()
	//reserved
	buffer.ReadUInt16()
	sampleRate := buffer.ReadUInt32() >> 16

	data := t.metaData.(*AudioMetaData)
	data.sampleRate = int(sampleRate)
	data.sampleBit = sampleSize
	data.channelCount = channelCount
	consume := buffer.ReadOffset() - offset
	return consume
}

func parseSTSDSubtitle(t *Track, size uint32, buffer utils.ByteBuffer) {

}

func parseSampleDescriptionBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stsd := sampleDescriptionBox{fullBox: fullBox{version: version, flags: flags}}
	stsd.entryCount = buffer.ReadUInt32()

	ret := len(data)
	for i := 0; i < int(stsd.entryCount); i++ {
		size := buffer.ReadUInt32()
		format := buffer.ReadUInt32()
		if size >= 16 {
			//reserved
			buffer.ReadUInt32()
			buffer.ReadUInt16()
			//data_reference_index
			_ = buffer.ReadUInt16()
		} else if size <= 7 {
			return nil, -1, fmt.Errorf("invalid data")
		}

		trak := ctx.tracks[len(ctx.tracks)-1]
		var ok bool
		codecId := utils.AVCodecIdNONE
		switch trak.MetaData().MediaType() {
		case utils.AVMediaTypeVideo:
			codecId, ok = videoTags[format]
			trak.metaData.setCodeId(codecId)
			if ok {
				if err := parseSTSDVideo(trak, size, buffer); err != nil {
					return nil, 0, err
				}
			}
			break
		case utils.AVMediaTypeAudio:
			codecId, ok = audioTags[format]
			trak.metaData.setCodeId(codecId)
			if ok {
				offset := buffer.ReadOffset()
				consume := parseSTSDAudio(trak, size, buffer)
				ret = offset + consume
			}
			break
		case utils.AVMediaTypeSubtitle:
			codecId, ok = subtitleTags[format]
			trak.metaData.setCodeId(codecId)
			if ok {
				parseSTSDSubtitle(trak, size, buffer)
			}
			break
		}

		if !ok {
			return nil, -1, fmt.Errorf("not find codec id of the %d format in sample entry", format)
		}

	}

	ctx.tracks[len(ctx.tracks)-1].mark |= markSampleDescription
	ctx.tracks[len(ctx.tracks)-1].stsd = &stsd
	return &stsd, ret, nil
}

func parseDecodingTimeToSampleBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stts := decodingTimeToSampleBox{fullBox: fullBox{version: version, flags: flags}}
	stts.entryCount = buffer.ReadUInt32()
	stts.sampleCount = make([]uint32, stts.entryCount)
	stts.sampleDelta = make([]uint32, stts.entryCount)
	for i := 0; i < int(stts.entryCount); i++ {
		stts.sampleCount[i] = buffer.ReadUInt32()
		stts.sampleDelta[i] = buffer.ReadUInt32()
	}
	ctx.tracks[len(ctx.tracks)-1].mark |= markTimeToSample
	ctx.tracks[len(ctx.tracks)-1].stts = &stts
	return &stts, len(data), nil
}

func parseCompositionTimeToSampleBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	ctts := compositionTimeToSampleBox{fullBox: fullBox{version: version, flags: flags}}

	ctts.entryCount = buffer.ReadUInt32()
	ctts.sampleCount = make([]uint32, ctts.entryCount)
	ctts.sampleOffset = make([]uint32, ctts.entryCount)
	for i := 0; i < int(ctts.entryCount); i++ {
		ctts.sampleCount[i] = buffer.ReadUInt32()
		if version == 0 {
			ctts.sampleCount[i] = buffer.ReadUInt32()
		} else if version == 1 {
			ctts.sampleOffset[i] = uint32(buffer.ReadInt32())
		}
	}

	return &ctts, len(data), nil
}

func parseCompositionToDecodeBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	cslg := compositionToDecodeBox{fullBox: fullBox{version: version, flags: flags}}
	if version == 0 {
		cslg.compositionToDTSShift = int64(buffer.ReadUInt32())
		cslg.leastDecodeToDisplayDelta = int64(buffer.ReadUInt32())
		cslg.greatestDecodeToDisplayDelta = int64(buffer.ReadUInt32())
		cslg.compositionStartTime = int64(buffer.ReadUInt32())
		cslg.compositionEndTime = int64(buffer.ReadUInt32())
	} else if version == 1 {
		cslg.compositionToDTSShift = buffer.ReadInt64()
		cslg.leastDecodeToDisplayDelta = buffer.ReadInt64()
		cslg.greatestDecodeToDisplayDelta = buffer.ReadInt64()
		cslg.compositionStartTime = buffer.ReadInt64()
		cslg.compositionEndTime = buffer.ReadInt64()
	}

	return &cslg, len(data), nil
}

func parseSampleToChunkBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stsc := sampleToChunkBox{fullBox: fullBox{version: version, flags: flags}}
	stsc.entryCount = buffer.ReadUInt32()

	stsc.firstChunk = make([]uint32, stsc.entryCount)
	stsc.samplesPerChunk = make([]uint32, stsc.entryCount)
	stsc.sampleDescriptionIndex = make([]uint32, stsc.entryCount)
	for i := 0; i < int(stsc.entryCount); i++ {
		stsc.firstChunk[i] = buffer.ReadUInt32()
		stsc.samplesPerChunk[i] = buffer.ReadUInt32()
		stsc.sampleDescriptionIndex[i] = buffer.ReadUInt32()
	}
	ctx.tracks[len(ctx.tracks)-1].mark |= markSampleToChunk
	ctx.tracks[len(ctx.tracks)-1].stsc = &stsc
	return &stsc, len(data), nil
}

func parseSampleSizeBoxes(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stsz := sampleSizeBox{fullBox: fullBox{version: version, flags: flags}}
	stsz.sampleSize = buffer.ReadUInt32()
	stsz.sampleCount = buffer.ReadUInt32()
	if stsz.sampleSize == 0 {
		stsz.entrySize = make([]uint32, stsz.sampleCount)
		for i := 0; i < int(stsz.sampleCount); i++ {
			stsz.entrySize[i] = buffer.ReadUInt32()
		}
	}
	ctx.tracks[len(ctx.tracks)-1].mark |= markSampleSize
	ctx.tracks[len(ctx.tracks)-1].stsz = &stsz
	return &stsz, len(data), nil
}

func parseCompactSampleSizeBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stz2 := compactSampleSizeBox{fullBox: fullBox{version: version, flags: flags}}
	stz2.fieldSize = buffer.ReadUInt8()
	stz2.sampleCount = buffer.ReadUInt32()
	for i := 0; i < int(stz2.sampleCount); i++ {
		//unsigned int(field_size) entry_size
		switch stz2.fieldSize {
		case 4:
			//entry[i]<<4	+	entry[i+1]
		case 8:
			buffer.ReadUInt8()
		case 16:
			buffer.ReadUInt16()
		}
	}

	return &stz2, len(data), nil
}

func parseChunkOffsetBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stco := chunkOffsetBox{fullBox: fullBox{version: version, flags: flags}}
	stco.entryCount = buffer.ReadUInt32()
	stco.chunkOffset = make([]uint32, 0, stco.entryCount)
	for i := 0; i < int(stco.entryCount); i++ {
		stco.chunkOffset = append(stco.chunkOffset, buffer.ReadUInt32())
	}
	ctx.tracks[len(ctx.tracks)-1].mark |= markChunkOffset
	ctx.tracks[len(ctx.tracks)-1].stco = &stco
	return &stco, len(data), nil
}

func parseChunkLargeOffsetBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	co64 := chunkLargeOffsetBox{fullBox: fullBox{version: version, flags: flags}}
	co64.entryCount = buffer.ReadUInt32()

	co64.chunkOffset = make([]uint64, co64.entryCount)
	for i := 0; i < int(co64.entryCount); i++ {
		co64.chunkOffset = append(co64.chunkOffset, buffer.ReadUInt64())
	}

	return &co64, len(data), nil
}

func parseSyncSampleBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stss := syncSampleBox{fullBox: fullBox{version: version, flags: flags}}

	stss.entryCount = buffer.ReadUInt32()
	//stss.sampleNumber = make([]uint32, stss.entryCount)
	stss.sampleNumber = make(map[uint32]byte, stss.entryCount)
	for i := 0; i < int(stss.entryCount); i++ {
		//stss.sampleNumber[i] = buffer.ReadUInt32()
		stss.sampleNumber[buffer.ReadUInt32()] = 0
	}

	ctx.tracks[len(ctx.tracks)-1].mark |= markSyncSample
	ctx.tracks[len(ctx.tracks)-1].stss = &stss
	return &stss, len(data), nil
}

func parseShadowSyncSampleBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stsh := shadowSyncSampleBox{fullBox: fullBox{version: version, flags: flags}}

	stsh.entryCount = buffer.ReadUInt32()
	for i := 0; i < int(stsh.entryCount); i++ {
		shadowedSampleNumber := buffer.ReadUInt32()
		syncSampleNumber := buffer.ReadUInt32()
		println(shadowedSampleNumber)
		println(syncSampleNumber)
	}

	return &stsh, len(data), nil
}

func parsePaddingBitsBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	padb := paddingBitsBox{fullBox: fullBox{version: version, flags: flags}}
	padb.sampleCount = buffer.ReadUInt32()
	for i := 0; i < int(padb.sampleCount+1)/2; i++ {
		//bit(1) reserved = 0;
		//bit(3) pad1;
		//bit(1) reserved = 0;
		//bit(3) pad2;
	}

	return &padb, len(data), nil
}

func parseDegradationPriorityBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	stdp := degradationPriorityBox{fullBox: fullBox{version: version, flags: flags}}
	return &stdp, len(data), nil
}

func parseIndependentAndDisposableSamplesBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	sdtp := independentAndDisposableSamplesBox{fullBox: fullBox{version: version, flags: flags}}

	//	for (i=0; i < sample_count; i++){
	//	unsigned int(2) is_leading;
	//	unsigned int(2) sample_depends_on;
	//	unsigned int(2) sample_is_depended_on;
	//	unsigned int(2) sample_has_redundancy;
	//}

	return &sdtp, len(data), nil
}

func parseSubSampleInformationBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	subs := subSampleInformationBox{fullBox: fullBox{version: version, flags: flags}}
	subs.entryCount = buffer.ReadUInt32()
	for i := 0; i < int(subs.entryCount); i++ {
		sample_delta := buffer.ReadUInt32()
		println(sample_delta)
		subsample_count := buffer.ReadUInt16()
		if subsample_count > 0 {
			for j := 0; j < int(subsample_count); j++ {
				if version == 1 {
					subsample_size := buffer.ReadUInt32()
					println(subsample_size)
				} else {
					subsample_size := buffer.ReadUInt16()
					println(subsample_size)
				}
				subsample_priority := buffer.ReadUInt8()
				discardable := buffer.ReadUInt8()
				codec_specific_parameters := buffer.ReadUInt32()
				println(subsample_priority)
				println(discardable)
				println(codec_specific_parameters)

			}
		}
	}

	return &subs, len(data), nil
}

func parseSampleAuxiliaryInformationSizesBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	saiz := sampleAuxiliaryInformationSizesBox{fullBox: fullBox{version: version, flags: flags}}
	if saiz.flags&0x1 != 0 {
		saiz.auxInfoType = buffer.ReadUInt32()
		saiz.auxInfoTypeParameter = buffer.ReadUInt32()
	}
	saiz.defaultSampleInfoSize = buffer.ReadUInt8()
	if saiz.defaultSampleInfoSize == 0 {
		saiz.sampleInfoSize = data[len(data)-8:]
	}

	return &saiz, len(data), nil
}

func parseSampleAuxiliaryInformationOffsetsBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	saio := sampleAuxiliaryInformationOffsetsBox{fullBox: fullBox{version: version, flags: flags}}
	if saio.flags&0x1 != 0 {
		saio.auxInfoType = buffer.ReadUInt32()
		saio.auxInfoTypeParameter = buffer.ReadUInt32()
	}
	saio.entryCount = buffer.ReadUInt32()

	return &saio, len(data), nil
}

func parseSampleToGroupBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	sbgp := sampleToGroupBox{fullBox: fullBox{version: version, flags: flags}}
	sbgp.groupingType = buffer.ReadUInt32()
	if sbgp.version == 1 {
		sbgp.groupingTypeParameter = buffer.ReadUInt32()
	}
	sbgp.entryCount = buffer.ReadUInt32()
	for i := 0; i < int(sbgp.entryCount); i++ {
		sbgp.sampleCount = append(sbgp.sampleCount, buffer.ReadUInt32())
		sbgp.groupDescriptionIndex = append(sbgp.groupDescriptionIndex, buffer.ReadUInt32())
	}

	return &sbgp, len(data), nil
}

func parseSampleGroupDescriptionBox(ctx *deMuxContext, data []byte) (box, int, error) {
	buffer := utils.NewByteBuffer(data)
	version := buffer.ReadUInt8()
	flags := buffer.ReadUInt24()
	sgpd := sampleGroupDescriptionBox{fullBox: fullBox{version: version, flags: flags}}
	sgpd.groupingType = buffer.ReadUInt32()
	if sgpd.version == 1 {
		sgpd.defaultLength = buffer.ReadUInt32()
	} else if version >= 2 {
		sgpd.defaultSampleDescriptionIndex = buffer.ReadUInt32()
	}

	sgpd.entryCount = buffer.ReadUInt32()
	for i := 0; i < int(sgpd.entryCount); i++ {
		if sgpd.version == 1 && sgpd.defaultLength == 0 {
			sgpd.descriptionLength = append(sgpd.descriptionLength, buffer.ReadUInt32())
		}
		//SampleGroupEntry (grouping_type);
		// an instance of a class derived from SampleGroupEntry
		// that is appropriate and permitted for the media type
	}

	return &sgpd, len(data), nil
}
