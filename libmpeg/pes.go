package libmpeg

import (
	"github.com/yangjiechina/avformat/libbufio"
)

const (
	StreamIdPrivateStream1 = 0xBD
	StreamIdPaddingStream  = 0xBE
	StreamIdPrivateStream2 = 0xBF
	StreamIdAudio          = 0xC0 //110x xxxx
	StreamIdVideo          = 0xE0 //1110 xxxx
	StreamIdH624           = 0xE2

	PesExistPtsMark    = 0x2
	PesExistPtsDtsMark = 0x3
)

type PESHeader struct {
	streamId     byte
	packetLength uint16

	//'10' 2 bslbf
	pesScramblingControl   byte //2
	pesPriority            byte //1
	dataAlignmentIndicator byte //1
	copyright              byte //1
	originalOrCopy         byte //1

	ptsDtsFlags            byte //2
	escrFlag               byte //1
	esRateFlag             byte //1
	dsmTrickModeFlag       byte //1
	additionalCopyInfoFlag byte //1
	pesCrcFlag             byte //1
	pesExtensionFlag       byte //1
	pesHeaderDataLength    byte //8

	escrBase      uint64
	escrExtension uint16 //9 bits
	esRate        uint32 //22 bits

	pts int64
	dts int64
}

func NewPESPacket(streamId byte) *PESHeader {
	return &PESHeader{
		streamId: streamId,
		pts:      -1,
		dts:      -1,
	}
}

func (p *PESHeader) Reset() {
	//p.streamId = 0
	p.packetLength = 0
	p.pesScramblingControl = 0
	p.pesPriority = 0
	//p.dataAlignmentIndicator = 0
	p.copyright = 0
	p.originalOrCopy = 0
	p.ptsDtsFlags = 0
	p.escrFlag = 0
	p.esRateFlag = 0
	p.dsmTrickModeFlag = 0
	p.additionalCopyInfoFlag = 0
	p.pesCrcFlag = 0
	p.pesExtensionFlag = 0
	p.pesHeaderDataLength = 0
	p.escrBase = 0
	p.escrExtension = 0
	p.esRate = 0
	//p.pts = -1
	//p.dts = -1
}

func (p *PESHeader) ToBytes(dst []byte) int {
	dst[0] = 0x00
	dst[1] = 0x00
	dst[2] = 0x01
	dst[3] = p.streamId

	dst[6] = 0x80
	dst[6] = dst[6] | p.pesScramblingControl<<4
	dst[6] = dst[6] | p.pesPriority<<3
	dst[6] = dst[6] | p.dataAlignmentIndicator<<2
	dst[6] = dst[6] | p.copyright<<1
	dst[6] = dst[6] | p.originalOrCopy

	dst[7] = p.ptsDtsFlags << 6
	dst[7] = dst[7] | p.escrFlag<<5
	dst[7] = dst[7] | p.esRateFlag<<4
	dst[7] = dst[7] | p.dsmTrickModeFlag<<3
	dst[7] = dst[7] | p.additionalCopyInfoFlag<<2
	dst[7] = dst[7] | p.pesCrcFlag<<1
	dst[7] = dst[7] | p.pesExtensionFlag

	//dst[8] = p.pesHeaderDataLength

	offset, temp := 9, 9
	if p.ptsDtsFlags&0x2 == 0x2 {
		//4bits
		dst[offset] = 0x20
		//PTS [32..30]
		dst[offset] = dst[offset] | (byte(p.pts>>30) << 1)
		//mark bit
		dst[offset] = dst[offset] | 0x1
		offset++
		dst[offset] = byte(p.pts >> 22)
		offset++
		dst[offset] = byte(p.pts >> 14)
		dst[offset] = dst[offset] | 0x1
		offset++
		dst[offset] = byte(p.pts >> 7)
		offset++
		dst[offset] = byte(p.pts) << 1
		dst[offset] = dst[offset] | 0x1

		offset++
	}

	if p.ptsDtsFlags&0x1 == 0x1 {
		dst[temp] = dst[temp] | 0x30

		//4bits `0001`
		dst[offset] = 0x10
		//PTS [32..30]
		dst[offset] = dst[offset] | (byte(p.dts>>30) << 1)
		//mark bit
		dst[offset] = dst[offset] | 0x1
		offset++
		dst[offset] = byte(p.dts >> 22)
		offset++
		dst[offset] = byte(p.dts >> 14)
		dst[offset] = dst[offset] | 0x1
		offset++
		dst[offset] = byte(p.dts >> 7)
		offset++
		dst[offset] = byte(p.dts) << 1
		dst[offset] = dst[offset] | 0x1

		offset++
	}

	p.pesHeaderDataLength = byte(offset - temp)
	dst[8] = p.pesHeaderDataLength
	return offset
}

func readPESHeader(p *PESHeader, src []byte) int {
	length := len(src)
	if length < 9 {
		return 0
	}

	p.streamId = src[3]
	p.packetLength = libbufio.BytesToUInt16(src[4], src[5])
	//1011 1100 1 program_stream_map
	//1011 1101 2 private_stream_1
	//1011 1110 padding_stream
	//1011 1111 3 private_stream_2
	//110x xxxx ISO/IEC 13818-3 or ISO/IEC 11172-3 or ISO/IEC 13818-7 or ISO/IEC 14496-3 audio stream number x xxxx
	//1110 xxxx ITU-T Rec. H.262 | ISO/IEC 13818-2 or ISO/IEC 11172-2 or ISO/IEC 14496-2 video stream number xxxx
	//1111 0000 3 ECM_stream
	//1111 0001 3 EMM_stream
	//1111 0010 5 ITU-T Rec. H.222.0 | ISO/IEC 13818-1 Annex A or ISO/IEC 13818- 6_DSMCC_stream
	//1111 0011 2 ISO/IEC_13522_stream
	//1111 0100 6 ITU-T Rec. H.222.1 type A
	//1111 0101 6 ITU-T Rec. H.222.1 type B
	//1111 0110 6 ITU-T Rec. H.222.1 type C
	//1111 0111 6 ITU-T Rec. H.222.1 type D
	//1111 1000 6 ITU-T Rec. H.222.1 type E
	//1111 1001 7 ancillary_stream
	//1111 1010 ISO/IEC14496-1_SL-packetized_stream
	//1111 1011 ISO/IEC14496-1_FlexMux_stream
	//1111 1100 … 1111 1110 reserved data stream
	//1111 1111 4 program_stream_directory

	//if (stream_id != program_stream_map
	//&& stream_id != padding_stream
	//&& stream_id != private_stream_2
	//&& stream_id != ECM
	//&& stream_id != EMM
	//&& stream_id != program_stream_directory
	//&& stream_id != DSMCC_stream
	//&& stream_id != ITU-T Rec. H.222.1 type E stream)

	if p.streamId != 0xBC && p.streamId != 0xBE && p.streamId != 0xBF && p.streamId != 0xF0 && p.streamId != 0xF1 && p.streamId != 0xff && p.streamId != 0xF2 && p.streamId != 0xF8 {

	} else {
		panic("Other unfinished")
	}
	p.pesScramblingControl = src[6] >> 4 & 0x3
	p.pesPriority = src[6] >> 3 & 0x1
	p.dataAlignmentIndicator = src[6] >> 2 & 0x1
	p.copyright = src[6] >> 1 & 0x1
	p.originalOrCopy = src[6] & 0x1
	p.ptsDtsFlags = src[7] >> 6 & 0x3
	p.escrFlag = src[7] >> 5 & 0x1
	p.esRateFlag = src[7] >> 4 & 0x1
	p.dsmTrickModeFlag = src[7] >> 3 & 0x1
	p.additionalCopyInfoFlag = src[7] >> 2 & 0x1
	p.pesCrcFlag = src[7] >> 1 & 0x1
	p.pesExtensionFlag = src[7] & 0x1
	p.pesHeaderDataLength = src[8]

	offset := 9
	if p.ptsDtsFlags&0x2 == 0x2 {
		p.pts = int64(src[offset]&0xE)<<29 | (int64(src[offset+1]) << 22) | (int64(src[offset+2]&0xFE) << 14) | (int64(src[offset+3]) << 7) | int64(src[offset+4]>>1)
		offset += 5
	}

	if p.ptsDtsFlags&0x1 == 0x1 {
		p.dts = int64(src[offset]&0xE)<<29 | (int64(src[offset+1]) << 22) | (int64(src[offset+2]&0xFE) << 14) | (int64(src[offset+3]) << 7) | int64(src[offset+4]>>1)
		offset += 5
	}

	if p.escrFlag == 0x1 {
		p.escrBase = (uint64(src[offset]&0x38) << 27) | (uint64(src[offset]&0x3) << 28) | (uint64(src[offset+1]) << 20) | (uint64(src[offset+2]&0xF8) << 12) | (uint64(src[offset+2]&0x3) << 13) | (uint64(src[offset+3]) << 5) | (uint64(src[offset+4] >> 3))
		p.escrExtension = uint16(src[offset+4]&0x3<<6) | uint16(src[offset+5]>>1)
		offset += 6
	}

	if p.esRateFlag == 0x1 {
		p.esRate = (uint32(src[offset]&0x7F) << 15) | (uint32(src[offset+1]) << 7) | uint32(src[offset+2]>>1)
		offset += 3
	}

	return offset
}
