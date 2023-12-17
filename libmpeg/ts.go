package libmpeg

const (
	PSIPAT = 0x0000
	PSICAT = 0x0001
	//PSINIT = 0x0000
	PSIPMT  = 0x0002
	PSITSDT = 0x0002

	TsPacketSize = 188
)

var stuffing [188]byte

func init() {
	for i := 0; i < len(stuffing); i++ {
		stuffing[i] = 0xFF
	}
}

type TSHeader struct {
	syncByte                   byte //1byte fixed 0x47
	transportErrorIndicator    byte //1bit
	payloadUnitStartIndicator  byte //1bit
	transportPriority          byte //1bit
	pid                        int  //13bits //0x0000-PAT/0x001-CAT/0x002-TSDT/0x0004-0x000F reserved/0x1FFF null packet
	transportScramblingControl byte //2bits
	adaptationFieldControl     byte //2bits 10/11/01/11
	continuityCounter          byte //4bits
}

func (h *TSHeader) toBytes(dst []byte) {
	dst[0] = 0x47
	dst[1] = (h.transportErrorIndicator & 0x1 << 7) | (h.payloadUnitStartIndicator & 0x1 << 6) | (h.transportPriority & 0x1 << 5) | byte(h.pid>>8&0x1F)
	dst[2] = byte(h.pid)
	dst[3] = (h.transportScramblingControl & 0x3 << 6) | (h.adaptationFieldControl & 0x3 << 4) | (h.continuityCounter & 0xF)
}

func readTSHeader(data []byte) (TSHeader, int) {
	h := TSHeader{}
	h.syncByte = data[0]
	h.transportErrorIndicator = data[1] >> 7 & 0x1
	h.payloadUnitStartIndicator = data[1] >> 6 & 0x1
	h.transportPriority = data[1] >> 5 & 0x1
	h.pid = int(data[1]&0x1F) << 8
	h.pid = h.pid | int(data[2])
	h.transportScramblingControl = data[3] >> 6 & 0x3
	h.adaptationFieldControl = data[3] >> 4 & 0x3
	h.continuityCounter = data[3] & 0xF
	index := 4

	switch h.adaptationFieldControl {
	case 0x00:
		//discard
		break
	case 0x01:
		break
	case 0x02:
		//adaptation field only,no payload
		break
	case 0x03:
		//2.4.3.4 adaptation_field
		length := data[4]
		index++
		index += int(length)
		break
	}

	return h, index
}

func readTableHeader(data []byte) int {
	//pointerField := data[0]
	//tableId := data[1]
	//sectionSyntaxIndicator := data[2] >> 7 & 0x01
	////'0'
	////2bits reserved
	sectionLength := int(data[2]&0xF) << 8
	sectionLength |= int(data[3])
	//transportStreamId := (int(data[4]) << 8) | int(data[5])
	////2bits reserved
	//versionNumber := data[6] >> 1 & 0x1F
	////1bit current_next_indicator
	//sectionNumber := data[7]
	//lastSectionNumber := data[8]
	//println(pointerField)
	//println(tableId)
	//println(sectionSyntaxIndicator)
	//println(transportStreamId)
	//println(versionNumber)
	//println(sectionNumber)
	//println(lastSectionNumber)

	return sectionLength
}

func readPAT(data []byte) []int {
	sectionLength := readTableHeader(data)
	index := 9
	sectionLength -= 5
	var pmt []int
	for sectionLength >= 8 {
		programNumber := (int(data[index]) << 8) | int(data[index+1])
		//reserved 3bits
		index += 2
		pid := (int(data[index]&0x1F) << 8) | int(data[index+1])
		if programNumber == 0 {
			//network pid
		} else {
			//pat
			pmt = append(pmt, pid)
		}
		index += 2
		sectionLength -= 4
	}

	return pmt
}

func readPMT(data []byte) []int {
	sectionLength := readTableHeader(data)
	index := 9
	sectionLength -= 5
	pcrPid := int(data[index]&0x1F) << 8
	index++
	pcrPid |= int(data[index])
	index++
	programInfoLength := int(data[index]&0xF) << 8
	index++
	programInfoLength |= int(data[index])
	index++

	var pid []int
	for sectionLength >= 9 {
		//streamType
		_ = data[index]
		index++
		elementaryPid := int(data[index]&0x1F) << 8
		index++
		elementaryPid |= int(data[index])
		index++
		esInfoLength := int(data[index]&0x1F) << 8
		pid = append(pid, elementaryPid)
		index++
		esInfoLength |= int(data[index])
		index++
		sectionLength -= 5
	}

	return pid
}

func readCASection(data []byte) int {

	return 0
}

func generatePAT(data []byte, counter byte) int {
	var n int
	header := TSHeader{0x47, 0, 0, 0, PSIPAT, 0, 0, counter}
	header.toBytes(data)
	n += 4
	//section 2.4.4.3
	//program_association_section() {
	//	table_id 8 uimsbf
	//	section_syntax_indicator 1 bslbf
	//	'0' 1 bslbf
	//	reserved 2 bslbf
	//	section_length 12 uimsbf
	//	transport_stream_id 16 uimsbf
	//	reserved 2 bslbf
	//	version_number 5 uimsbf
	//	current_next_indicator 1 bslbf
	//	section_number 8 uimsbf
	//	last_section_number 8 uimsbf
	//	for (i = 0; i < N; i++) {
	//	program_number 16 uimsbf
	//	reserved 3 bslbf
	//	if (program_number = = '0') {
	//	ISO/IEC 13818-1：2007(C)
	//	44 ITU-T H.222.0建议书 (05/2006)
	//	表 2-30－节目相关分段
	//	句 法 比 特 数 助 记 符
	//	network_PID 13 uimsbf
	//	}
	//	else {
	//	program_map_PID 13 uimsbf
	//	}
	//	}
	//	CRC_32 32 rpchof
	//}
	var sectionLength int16
	var transportStreamId int64
	//PAT发生变化时，版本号发生变化
	//currentNextIndicator 为1时，当前PAT有效，为0时，下个PAT才生效
	var versionNumber byte
	var currentNextIndicator byte

	//PAT的序号，第一个PAT必须要从0开始
	var sectionNumber byte
	var lastSectionNumber byte

	sectionLength = 8

	data[n] = 0x00
	n++
	data[n] = 0x80 | (byte(sectionLength>>8) & 0x3)
	n++
	data[n] = byte(sectionLength)
	n++
	data[n] = byte(transportStreamId >> 8)
	n++
	data[n] = byte(transportStreamId)
	n++
	data[n] = (versionNumber << 6) | (currentNextIndicator & 0x1)
	n++
	data[n] = sectionNumber
	n++
	data[n] = lastSectionNumber
	n++

	//program_number 为0时，后面必须时network_PID. 此处封装PMT_ID
	data[n] = 0
	n++
	data[n] = 0x1
	n++

	data[n] = PSIPMT >> 8 & 0x10
	n++
	data[n] = PSIPMT & 0xFFFF
	n++

	return n
}

func generatePMT(data []byte, counter byte, streamTypes [][2]int16) int {
	header := TSHeader{0x47, 0, 0, 0, PSIPMT, 0, 0, counter}
	header.toBytes(data)
	var n int
	n += 4
	//section 2.4.4.8
	var sectionLength int16
	var programNumber int16
	var versionNumber byte
	var currentNextIndicator byte
	var sectionNumber byte
	var lastSectionNumber byte
	var pcrPid int16
	var infoLength int16

	data[n] = PSIPMT
	n++
	data[n] = (1 << 7) | byte(sectionLength>>8&0x3)
	n++
	data[n] = byte(sectionLength)
	n++

	data[n] = byte(programNumber >> 8)
	n++
	data[n] = byte(programNumber)
	n++

	data[n] = (versionNumber & 0x10 << 1) | (currentNextIndicator & 0x1)
	n++

	data[n] = sectionNumber
	n++
	data[n] = lastSectionNumber
	n++
	data[n] = byte(pcrPid & 0x10)
	n++
	data[n] = byte(pcrPid)
	n++

	data[n] = byte(infoLength >> 8 & 0x3)
	n++
	data[n] = byte(infoLength)
	n++

	if infoLength > 0 {

	}

	n += int(infoLength)

	for _, streamType := range streamTypes {
		data[n] = byte(streamType[0])
		n++
		//elementary_PID
		//负载PES的TS包的PID
		data[n] = byte(streamType[1] & 0x10)
		n++
		data[n] = byte(streamType[1])
		n++

		data[n] = 0x0
		n++
		data[n] = 0x0
		n++
	}

	return n
}
