package libmpeg

import (
	"encoding/binary"
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
	"math"
)

// section 2.4.4
const (
	PsiPat = 0x0000

	// PsiPmt 在PAT中自定义
	PsiPmt = 0x1000

	// PsiCat 私有数据
	PsiCat = 0x0001 //私有数据
	// PsiTSdt 描述流
	PsiTSdt = 0x0002

	// PsiNit 网络信息
	PsiNit = 0x0000

	TableIdPAS = 0x00

	TableIdCAS = 0x01

	TableIdPMS = 0x2

	TsPacketSize = 188

	TsPacketStartPid = 0x100
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
	payloadUnitStartIndicator  byte //1bit //1-PSI(PAT/PMT/...)和PES开始包
	transportPriority          byte //1bit
	pid                        int  //13bits //0x0000-PAT/0x001-CAT/0x002-TSDT/0x0004-0x000F reserved/0x1FFF null packet
	transportScramblingControl byte //2bits 加密使用
	adaptationFieldControl     byte //2bits 10-全是自适应字段数据 11-自适应字段后跟随负载数据 01-全是负载数据
	continuityCounter          byte //4bits
	adaptationField            AdaptationField
}

type AdaptationField struct {
	length byte

	//discontinuity_indicator 1 bslbf
	//random_access_indicator 1 bslbf
	//elementary_stream_priority_indicator 1 bslbf
	//PCR_flag 1 bslbf
	//OPCR_flag 1 bslbf
	//splicing_point_flag 1 bslbf
	//transport_private_data_flag 1 bslbf
	//adaptation_field_extension_flag 1 bslbf
	tag             byte
	pcr             int64
	opcr            int64
	spliceCountdown byte
	privateData     []byte
	stuffingCount   int

	//adaptation_field_extension_length
}

func NewTSHeader(pid int, start, counter byte) TSHeader {
	return TSHeader{
		syncByte:                   0x47,
		transportErrorIndicator:    0x0,
		payloadUnitStartIndicator:  start,
		transportPriority:          0x0,
		pid:                        pid,
		transportScramblingControl: 0,
		adaptationFieldControl:     0x01,
		continuityCounter:          counter,
		adaptationField:            AdaptationField{length: 0xFF},
	}
}

// section 2.4.3.2
func (h *TSHeader) toBytes(dst []byte) {
	dst[0] = 0x47
	dst[1] = (h.transportErrorIndicator & 0x1 << 7) | (h.payloadUnitStartIndicator & 0x1 << 6) | (h.transportPriority & 0x1 << 5) | byte(h.pid>>8&0x1F)
	dst[2] = byte(h.pid)
	dst[3] = (h.transportScramblingControl & 0x3 << 6) | (h.adaptationFieldControl & 0x3 << 4) | (h.continuityCounter & 0xF)

	if h.adaptationFieldControl == 0x3 && h.adaptationField.length > 0 {
		dst[4] = h.adaptationField.length
		dst[5] = h.adaptationField.tag
	}

	//重置为没有自适应数据
	h.adaptationFieldControl = 0x1
	h.adaptationField.length = 0
	h.adaptationField.tag = 0
}

func (h *TSHeader) increaseCounter() {
	h.continuityCounter = (h.continuityCounter + 1) % 16
}

func (h *TSHeader) writePCR(data []byte, pcr int64) int {
	h.adaptationFieldControl = 0x3
	n := h.writeAdaptationField(data[1:], pcr)
	h.adaptationField.length = byte(n)
	//预留adaptation_field_length字段
	return 1 + n
}

// PES数据不足以填满整个TS包时，填充FF
func (h *TSHeader) fill(data []byte, count int) int {
	h.adaptationFieldControl = 0x3

	//如果还没有写过自适应字段
	//预留2字节
	var n int
	if h.adaptationField.length < 1 {
		count = libbufio.MaxInt(count-2, 0)
		n = 2
		h.adaptationField.length++
	}

	copy(data[n:], stuffing[:count])
	h.adaptationField.length += byte(count)
	return n + count
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
func writePAT(data []byte, counter byte) int {
	var n int
	header := NewTSHeader(PsiPat, 1, counter)
	header.toBytes(data)
	n += 4
	//高2位必须为0
	var sectionLength int16
	var transportStreamId int64
	//PAT发生变化时，版本号发生变化
	//currentNextIndicator 为1时，当前PAT有效，为0时，下个PAT才生效
	var versionNumber byte
	var currentNextIndicator byte

	//PAT的序号，第一个PAT必须要从0开始
	var sectionNumber byte
	var lastSectionNumber byte

	currentNextIndicator = 1
	sectionLength = 13

	data[n] = 0x00
	n++
	data[n] = TableIdPAS
	n++
	data[n] = 0x80 | (byte(sectionLength>>8) & 0x03)
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

	data[n] = PsiPmt >> 8 & 0x10
	n++
	data[n] = PsiPmt & 0xFF
	n++

	//crc32
	crc32 := utils.CalculateCrcMpeg2(data[5:n])
	binary.BigEndian.PutUint32(data[n:], crc32)
	n += 4
	return n
}

func writePMT(data []byte, counter byte, programNumber_, pcrPid_ int, streamTypes [][2]int16) int {
	header := NewTSHeader(PsiPmt, 1, counter)
	header.toBytes(data)
	var n int
	n += 4
	//section 2.4.4.8
	var sectionLength int16
	//整个ts流中作为关联PMT的序号
	var programNumber int16
	var versionNumber byte
	var currentNextIndicator byte
	//必须为0
	var sectionNumber byte
	//必须为0
	var lastSectionNumber byte
	//programNumber指向包含PCR的TS包的PID
	var pcrPid int16
	var infoLength int16

	currentNextIndicator = 1
	programNumber = int16(programNumber_)
	pcrPid = int16(pcrPid_)

	data[n] = 0x00
	n++
	data[n] = TableIdPMS
	n++
	data[n] = (1 << 7) | byte(sectionLength>>8&0x03)
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
	data[n] = byte(pcrPid >> 8 & 0x1F)
	n++
	data[n] = byte(pcrPid & 0xFF)
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
		data[n] = byte(streamType[1] >> 8 & 0x1F)
		n++
		data[n] = byte(streamType[1] & 0xFF)
		n++

		data[n] = 0x0
		n++
		data[n] = 0x0
		n++
	}

	//sectionLength
	data[6] = (1 << 7) | byte((n-4)>>8&0x03)
	data[7] = byte(n - 4)

	crc32 := utils.CalculateCrcMpeg2(data[5:n])
	binary.BigEndian.PutUint32(data[n:], crc32)
	n += 4
	return n
}

// section 2.4.3.4
func (h *TSHeader) writeAdaptationField(data []byte, pcr int64) int {
	var n int

	h.adaptationField.tag = 0x1 << 4

	//var pcr int64
	var opcr int64
	var spliceCountdown byte
	var privateData []byte

	data[n] = h.adaptationField.tag
	n++
	//MPEG-2标准中, 时钟频率为27MHZ
	//PES中的DTS和PTS 90KHZ
	if h.adaptationField.tag>>4&0x1 == 1 {
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

	if h.adaptationField.tag>>3&0x1 == 1 {
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

	if h.adaptationField.tag>>2&0x1 == 1 {
		data[n] = spliceCountdown
		n++
	}

	if h.adaptationField.tag>>1&0x1 == 1 {
		utils.Assert(len(privateData) > 0)

		data[n] = byte(len(privateData))
		n++
		copy(data[n:], privateData)
		n += len(privateData)
	}

	//暂不实现扩展字段
	utils.Assert(h.adaptationField.tag&0x1 == 0)
	return n
}

func writeAud(data []byte, id utils.AVCodecID) int {
	binary.BigEndian.PutUint32(data, 0x1)
	if utils.AVCodecIdH264 == id {
		data[4] = 0x09
		data[5] = 0xF0
		return 6
	} else if utils.AVCodecIdH265 == id {
		data[4] = 0x46
		data[5] = 0x01
		data[6] = 0x50
		return 7
	} else if utils.AVCodecIdH265 == id {
		data[4] = 0x00
		data[5] = 0xA1
		data[6] = 0x28
		return 7
	}

	utils.Assert(false)
	return -1
}
