package libmpeg

import (
	"github.com/yangjiechina/avformat/utils"
	"math"
)

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

// section 2.4.4.3
func generatePAT(data []byte, counter byte) int {
	var n int
	header := TSHeader{0x47, 0, 0, 0, PSIPAT, 0, 0, counter}
	header.toBytes(data)
	n += 4
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

// section 2.4.3.4
func generateAdaptationField(data []byte, pcr int64) int {
	var n int
	n++

	var discontinuityIndicator byte
	var randomAccessIndicator byte
	var elementaryStreamPriorityIndicator byte
	var PCRFlag byte
	var OPCRFlag byte
	var splicingPointFlag byte
	var transportPrivateDataFlag byte
	var adaptationFieldExtensionFlag byte

	//var pcr int64
	var opcr int64
	var spliceCountdown byte
	var privateData []byte

	discontinuityIndicator = 0
	//当前和后续TS包PID可能相同
	randomAccessIndicator = 1
	elementaryStreamPriorityIndicator = 0
	PCRFlag = 1
	OPCRFlag = 0
	splicingPointFlag = 0
	transportPrivateDataFlag = 0
	adaptationFieldExtensionFlag = 0
	data[n] = (discontinuityIndicator & 0x1 << 7) | (randomAccessIndicator & 0x1 << 6) | (elementaryStreamPriorityIndicator & 0x1 << 5) | (PCRFlag & 0x1 << 4) | (OPCRFlag & 0x1 << 3) | (splicingPointFlag & 0x1 << 2) | (transportPrivateDataFlag & 0x1 << 1) | (adaptationFieldExtensionFlag & 0x1)
	n++
	//MPEG-2标准中, 时钟频率为27MHZ
	//PES中的DTS和PTS 90KHZ
	if PCRFlag&0x1 == 1 {
		//不能超过42位
		utils.Assert(pcr <= 0x3FFFFFFFFF)
		//PCR(i)=PCR base(i)x300+ PCR ext(i)
		//PCR base(i) = ((system clock frequency x t(i))DIV 300) % 233
		//PCR ext(i) = ((system clock frequency x t(i)) DIV 1) % 300
		pcrBase := pcr / 300 % int64(math.Pow(2, 33))
		pcrExtension := pcr % 300

		data[n] = byte(pcrBase >> 25 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 17 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 9 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 1 & 0xFF)
		n++
		data[n] = byte(pcrBase&0x1<<7) | byte(pcrExtension>>8&0x1)
		n++
		data[n] = byte(pcrExtension & 0xFF)
		n++
	}

	if OPCRFlag&0x1 == 1 {
		pcrBase := opcr / 300 % int64(math.Pow(2, 33))
		pcrExtension := opcr % 300

		data[n] = byte(pcrBase >> 25 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 17 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 9 & 0xFF)
		n++
		data[n] = byte(pcrBase >> 1 & 0xFF)
		n++
		data[n] = byte(pcrBase&0x1<<7) | byte(pcrExtension>>8&0x1)
		n++
		data[n] = byte(pcrExtension & 0xFF)
		n++
	}

	if splicingPointFlag&0x1 == 1 {
		data[n] = spliceCountdown
		n++
	}

	if transportPrivateDataFlag&0x1 == 1 {
		utils.Assert(len(privateData) > 0)

		data[n] = byte(len(privateData))
		n++
		copy(data[n:], privateData)
		n += len(privateData)
	}
	return n
}
