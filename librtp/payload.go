package librtp

import "github.com/yangjiechina/avformat/utils"

//PT   encoding    media type  clock rate   channels
//name                    (Hz)
//___________________________________________________
//0    PCMU        A            8,000       1
//1    reserved    A
//2    reserved    A
//3    GSM         A            8,000       1
//4    G723        A            8,000       1
//5    DVI4        A            8,000       1
//6    DVI4        A           16,000       1
//7    LPC         A            8,000       1
//8    PCMA        A            8,000       1
//9    G722        A            8,000       1
//10   L16         A           44,100       2
//11   L16         A           44,100       1
//12   QCELP       A            8,000       1
//13   CN          A            8,000       1
//14   MPA         A           90,000       (see text)
//15   G728        A            8,000       1
//16   DVI4        A           11,025       1
//17   DVI4        A           22,050       1
//18   G729        A            8,000       1
//19   reserved    A
//20   unassigned  A
//21   unassigned  A
//22   unassigned  A
//23   unassigned  A
//dyn  G726-40     A            8,000       1
//dyn  G726-32     A            8,000       1
//dyn  G726-24     A            8,000       1
//dyn  G726-16     A            8,000       1
//dyn  G729D       A            8,000       1
//dyn  G729E       A            8,000       1
//dyn  GSM-EFR     A            8,000       1
//dyn  L8          A            var.        var.
//dyn  RED         A                        (see text)
//dyn  VDVI        A            var.        1
//
//PT   encoding    media type  clock rate   channels
//name                    (Hz)
//___________________________________________________
//0    PCMU        A            8,000       1
//1    reserved    A
//2    reserved    A
//3    GSM         A            8,000       1
//4    G723        A            8,000       1
//5    DVI4        A            8,000       1
//6    DVI4        A           16,000       1
//7    LPC         A            8,000       1
//8    PCMA        A            8,000       1
//9    G722        A            8,000       1
//10   L16         A           44,100       2
//11   L16         A           44,100       1
//12   QCELP       A            8,000       1
//13   CN          A            8,000       1
//14   MPA         A           90,000       (see text)
//15   G728        A            8,000       1
//16   DVI4        A           11,025       1
//17   DVI4        A           22,050       1
//18   G729        A            8,000       1
//19   reserved    A
//20   unassigned  A
//21   unassigned  A
//22   unassigned  A
//23   unassigned  A
//dyn  G726-40     A            8,000       1
//dyn  G726-32     A            8,000       1
//dyn  G726-24     A            8,000       1
//dyn  G726-16     A            8,000       1
//dyn  G729D       A            8,000       1
//dyn  G729E       A            8,000       1
//dyn  GSM-EFR     A            8,000       1
//dyn  L8          A            var.        var.
//dyn  RED         A                        (see text)
//dyn  VDVI        A            var.        1
//
//Table 4: Payload types (PT) for audio encodings
//
//PT      encoding    media type  clock rate
//name                    (Hz)
//_____________________________________________
//24      unassigned  V
//25      CelB        V           90,000
//26      JPEG        V           90,000
//27      unassigned  V
//28      nv          V           90,000
//29      unassigned  V
//30      unassigned  V
//31      H261        V           90,000
//32      MPV         V           90,000
//33      MP2T        AV          90,000
//34      H263        V           90,000
//35-71   unassigned  ?
//72-76   reserved    N/A         N/A
//77-95   unassigned  ?
//96-127  dynamic     ?
//dyn     H263-1998   V           90,000

var (
	payloadTypes map[int]payloadType
)

type payloadType struct {
	pt        int
	encoding  string
	mediaType utils.AVMediaType
	codeId    utils.AVCodecID
	clockRate int
	channels  int
}

func init() {
	payloadTypes = map[int]payloadType{
		0:  {0, "PCMU", utils.AVMediaTypeAudio, utils.AVCodecIdPCMMULAW, 8000, 1},
		3:  {3, "GSM", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		4:  {4, "G723", utils.AVMediaTypeAudio, utils.AVCodecIdG7231, 8000, 1},
		5:  {5, "DVI4", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		6:  {6, "DVI4", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 16000, 1},
		7:  {7, "LPC", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		8:  {8, "PCMA", utils.AVMediaTypeAudio, utils.AVCodecIdPCMALAW, 8000, 1},
		9:  {9, "G722", utils.AVMediaTypeAudio, utils.AVCodecIdADPCMG722, 8000, 1},
		10: {10, "L16", utils.AVMediaTypeAudio, utils.AVCodecIdPCMS16BE, 44100, 2},
		11: {11, "L16", utils.AVMediaTypeAudio, utils.AVCodecIdPCMS16BE, 44100, 1},
		12: {12, "QCELP", utils.AVMediaTypeAudio, utils.AVCodecIdQCELP, 8000, 1},
		13: {13, "CN", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		14: {14, "MPA", utils.AVMediaTypeAudio, utils.AVCodecIdMP2, -1, -1},
		//14: {14, "MPA", utils.AVMediaTypeAudio, utils.AVCodecIdMP3, -1, -1},
		15: {15, "G728", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		16: {16, "DVI4", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 11025, 1},
		17: {17, "DVI4", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 22050, 1},
		18: {18, "G729", utils.AVMediaTypeAudio, utils.AVCodecIdNONE, 8000, 1},
		25: {25, "CelB", utils.AVMediaTypeVideo, utils.AVCodecIdNONE, 90000, -1},
		26: {26, "JPEG", utils.AVMediaTypeVideo, utils.AVCodecIdMJPEG, 90000, -1},
		28: {28, "nv", utils.AVMediaTypeVideo, utils.AVCodecIdNONE, 90000, -1},
		31: {31, "H261", utils.AVMediaTypeVideo, utils.AVCodecIdH261, 90000, -1},
		32: {32, "MPV", utils.AVMediaTypeVideo, utils.AVCodecIdMPEG1VIDEO, 90000, -1},
		//32: {32, "MPV", utils.AVMediaTypeVideo, utils.AVCodecIdMPEG2VIDEO, 90000, -1},
		33: {33, "MP2T", utils.AVMediaTypeData, utils.AVCodecIdMPEG2TS, 90000, -1},
		34: {34, "H263", utils.AVMediaTypeVideo, utils.AVCodecIdH263, 90000, -1},

		96: {96, "", utils.AVMediaTypeVideo, utils.AVCodecIdNONE, 90000, -1},
	}
}

type Profile struct {
}
