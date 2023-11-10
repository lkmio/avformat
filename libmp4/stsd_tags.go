package libmp4

import "github.com/yangjiechina/avformat/utils"

// ffmpeg用的小端字序
func mkTag(byte1, byte2, byte3, byte4 byte) uint32 {
	return (uint32(byte1) << 24) | (uint32(byte2) << 16) | (uint32(byte3) << 8) | uint32(byte4)
}

var (
	videoTags    map[uint32]utils.AVCodecID
	audioTags    map[uint32]utils.AVCodecID
	subtitleTags map[uint32]utils.AVCodecID
	dataTags     map[uint32]utils.AVCodecID
)

func init() {
	videoTags = map[uint32]utils.AVCodecID{
		mkTag('r', 'a', 'w', ' '): utils.AVCodecIdRAWVIDEO, /* uncompressed RGB */
		mkTag('y', 'u', 'v', '2'): utils.AVCodecIdRAWVIDEO, /* uncompressed YUV422 */
		mkTag('2', 'v', 'u', 'y'): utils.AVCodecIdRAWVIDEO, /* uncompressed 8-bit 4:2:2 */
		mkTag('y', 'u', 'v', 's'): utils.AVCodecIdRAWVIDEO, /* same as 2VUY but byte-swapped */

		mkTag('L', '5', '5', '5'): utils.AVCodecIdRAWVIDEO,
		mkTag('L', '5', '6', '5'): utils.AVCodecIdRAWVIDEO,
		mkTag('B', '5', '6', '5'): utils.AVCodecIdRAWVIDEO,
		mkTag('2', '4', 'B', 'G'): utils.AVCodecIdRAWVIDEO,
		mkTag('B', 'G', 'R', 'A'): utils.AVCodecIdRAWVIDEO,
		mkTag('R', 'G', 'B', 'A'): utils.AVCodecIdRAWVIDEO,
		mkTag('A', 'B', 'G', 'R'): utils.AVCodecIdRAWVIDEO,
		mkTag('b', '1', '6', 'g'): utils.AVCodecIdRAWVIDEO,
		mkTag('b', '4', '8', 'r'): utils.AVCodecIdRAWVIDEO,
		mkTag('b', '6', '4', 'a'): utils.AVCodecIdRAWVIDEO,
		mkTag('b', 'x', 'b', 'g'): utils.AVCodecIdRAWVIDEO, /* BOXX */
		mkTag('b', 'x', 'r', 'g'): utils.AVCodecIdRAWVIDEO,
		mkTag('b', 'x', 'y', 'v'): utils.AVCodecIdRAWVIDEO,
		mkTag('N', 'O', '1', '6'): utils.AVCodecIdRAWVIDEO,
		mkTag('D', 'V', 'O', 'O'): utils.AVCodecIdRAWVIDEO, /* Digital Voodoo SD 8 Bit */
		mkTag('R', '4', '2', '0'): utils.AVCodecIdRAWVIDEO, /* Radius DV YUV PAL */
		mkTag('R', '4', '1', '1'): utils.AVCodecIdRAWVIDEO, /* Radius DV YUV NTSC */

		mkTag('R', '1', '0', 'k'): utils.AVCodecIdR10K, /* uncompressed 10-bit RGB */
		mkTag('R', '1', '0', 'g'): utils.AVCodecIdR10K, /* uncompressed 10-bit RGB */
		mkTag('r', '2', '1', '0'): utils.AVCodecIdR210, /* uncompressed 10-bit RGB */
		mkTag('A', 'V', 'U', 'I'): utils.AVCodecIdAVUI, /* AVID Uncompressed deinterleaved UYVY422 */
		mkTag('A', 'V', 'r', 'p'): utils.AVCodecIdAVRP, /* Avid 1:1 10-bit RGB Packer */
		mkTag('S', 'U', 'D', 'S'): utils.AVCodecIdAVRP, /* Avid DS Uncompressed */
		mkTag('v', '2', '1', '0'): utils.AVCodecIdV210, /* uncompressed 10-bit 4:2:2 */
		mkTag('b', 'x', 'y', '2'): utils.AVCodecIdV210, /* BOXX 10-bit 4:2:2 */
		mkTag('v', '3', '0', '8'): utils.AVCodecIdV308, /* uncompressed  8-bit 4:4:4 */
		mkTag('v', '4', '0', '8'): utils.AVCodecIdV408, /* uncompressed  8-bit 4:4:4:4 */
		mkTag('v', '4', '1', '0'): utils.AVCodecIdV410, /* uncompressed 10-bit 4:4:4 */
		mkTag('Y', '4', '1', 'P'): utils.AVCodecIdY41P, /* uncompressed 12-bit 4:1:1 */
		mkTag('y', 'u', 'v', '4'): utils.AVCodecIdYUV4, /* libquicktime packed yuv420p */
		mkTag('Y', '2', '1', '6'): utils.AVCodecIdTARGAY216,

		mkTag('j', 'p', 'e', 'g'): utils.AVCodecIdMJPEG,  /* PhotoJPEG */
		mkTag('m', 'j', 'p', 'a'): utils.AVCodecIdMJPEG,  /* Motion-JPEG (format A) */
		mkTag('A', 'V', 'D', 'J'): utils.AVCodecIdMJPEG,  /* MJPEG with alpha-channel (AVID JFIF meridien compressed) */
		mkTag('A', 'V', 'R', 'n'): utils.AVCodecIdMJPEG,  /* MJPEG with alpha-channel (AVID ABVB/Truevision NuVista) */
		mkTag('d', 'm', 'b', '1'): utils.AVCodecIdMJPEG,  /* Motion JPEG OpenDML */
		mkTag('m', 'j', 'p', 'b'): utils.AVCodecIdMJPEGB, /* Motion-JPEG (format B) */

		mkTag('S', 'V', 'Q', '1'): utils.AVCodecIdSVQ1, /* Sorenson Video v1 */
		mkTag('s', 'v', 'q', '1'): utils.AVCodecIdSVQ1, /* Sorenson Video v1 */
		mkTag('s', 'v', 'q', 'i'): utils.AVCodecIdSVQ1, /* Sorenson Video v1 (from QT specs)*/
		mkTag('S', 'V', 'Q', '3'): utils.AVCodecIdSVQ3, /* Sorenson Video v3 */

		mkTag('m', 'p', '4', 'v'): utils.AVCodecIdMPEG4,
		mkTag('D', 'I', 'V', 'X'): utils.AVCodecIdMPEG4, /* OpenDiVX */ /* sample files at http://heroinewarrior.com/xmovie.php3 use this tag */
		mkTag('X', 'V', 'I', 'D'): utils.AVCodecIdMPEG4,
		mkTag('3', 'I', 'V', '2'): utils.AVCodecIdMPEG4, /* experimental: 3IVX files before ivx D4 4.5.1 */

		mkTag('h', '2', '6', '3'): utils.AVCodecIdH263, /* H.263 */
		mkTag('s', '2', '6', '3'): utils.AVCodecIdH263, /* H.263 ?? works */

		mkTag('d', 'v', 'c', 'p'): utils.AVCodecIdDVVIDEO, /* DV PAL */
		mkTag('d', 'v', 'c', ' '): utils.AVCodecIdDVVIDEO, /* DV NTSC */
		mkTag('d', 'v', 'p', 'p'): utils.AVCodecIdDVVIDEO, /* DVCPRO PAL produced by FCP */
		mkTag('d', 'v', '5', 'p'): utils.AVCodecIdDVVIDEO, /* DVCPRO50 PAL produced by FCP */
		mkTag('d', 'v', '5', 'n'): utils.AVCodecIdDVVIDEO, /* DVCPRO50 NTSC produced by FCP */
		mkTag('A', 'V', 'd', 'v'): utils.AVCodecIdDVVIDEO, /* AVID DV */
		mkTag('A', 'V', 'd', '1'): utils.AVCodecIdDVVIDEO, /* AVID DV100 */
		mkTag('d', 'v', 'h', 'q'): utils.AVCodecIdDVVIDEO, /* DVCPRO HD 720p50 */
		mkTag('d', 'v', 'h', 'p'): utils.AVCodecIdDVVIDEO, /* DVCPRO HD 720p60 */
		mkTag('d', 'v', 'h', '1'): utils.AVCodecIdDVVIDEO,
		mkTag('d', 'v', 'h', '2'): utils.AVCodecIdDVVIDEO,
		mkTag('d', 'v', 'h', '4'): utils.AVCodecIdDVVIDEO,
		mkTag('d', 'v', 'h', '5'): utils.AVCodecIdDVVIDEO, /* DVCPRO HD 50i produced by FCP */
		mkTag('d', 'v', 'h', '6'): utils.AVCodecIdDVVIDEO, /* DVCPRO HD 60i produced by FCP */
		mkTag('d', 'v', 'h', '3'): utils.AVCodecIdDVVIDEO, /* DVCPRO HD 30p produced by FCP */

		mkTag('V', 'P', '3', '1'): utils.AVCodecIdVP3,     /* On2 VP3 */
		mkTag('r', 'p', 'z', 'a'): utils.AVCodecIdRPZA,    /* Apple Video (RPZA) */
		mkTag('c', 'v', 'i', 'd'): utils.AVCodecIdCINEPAK, /* Cinepak */
		mkTag('8', 'B', 'P', 'S'): utils.AVCodecId8BPS,    /* Planar RGB (8BPS) */
		mkTag('s', 'm', 'c', ' '): utils.AVCodecIdSMC,     /* Apple Graphics (SMC) */
		mkTag('r', 'l', 'e', ' '): utils.AVCodecIdQTRLE,   /* Apple Animation (RLE) */
		mkTag('r', 'l', 'e', '1'): utils.AVCodecIdSGIRLE,  /* SGI RLE 8-bit */
		mkTag('W', 'R', 'L', 'E'): utils.AVCodecIdMSRLE,
		mkTag('q', 'd', 'r', 'w'): utils.AVCodecIdQDRAW,   /* QuickDraw */
		mkTag('Q', 'k', 'B', 'k'): utils.AVCodecIdCDTOONS, /* CDToons */

		mkTag('W', 'R', 'A', 'W'): utils.AVCodecIdRAWVIDEO,

		mkTag('h', 'e', 'v', '1'): utils.AVCodecIdHEVC, /* HEVC/H.265 which indicates parameter sets may be in ES */
		mkTag('h', 'v', 'c', '1'): utils.AVCodecIdHEVC, /* HEVC/H.265 which indicates parameter sets shall not be in ES */
		mkTag('d', 'v', 'h', 'e'): utils.AVCodecIdHEVC, /* HEVC-based Dolby Vision derived from hev1 */

		mkTag('a', 'v', 'c', '1'): utils.AVCodecIdH264, /* aCd/H.264 */
		mkTag('a', 'v', 'c', '2'): utils.AVCodecIdH264,
		mkTag('a', 'v', 'c', '3'): utils.AVCodecIdH264,
		mkTag('a', 'v', 'c', '4'): utils.AVCodecIdH264,
		mkTag('a', 'i', '5', 'p'): utils.AVCodecIdH264, /* AVC-Intra  50M 720p24/30/60 */
		mkTag('a', 'i', '5', 'q'): utils.AVCodecIdH264, /* AVC-Intra  50M 720p25/50 */
		mkTag('a', 'i', '5', '2'): utils.AVCodecIdH264, /* AVC-Intra  50M 1080p25/50 */
		mkTag('a', 'i', '5', '3'): utils.AVCodecIdH264, /* AVC-Intra  50M 1080p24/30/60 */
		mkTag('a', 'i', '5', '5'): utils.AVCodecIdH264, /* AVC-Intra  50M 1080i50 */
		mkTag('a', 'i', '5', '6'): utils.AVCodecIdH264, /* AVC-Intra  50M 1080i60 */
		mkTag('a', 'i', '1', 'p'): utils.AVCodecIdH264, /* AVC-Intra 100M 720p24/30/60 */
		mkTag('a', 'i', '1', 'q'): utils.AVCodecIdH264, /* AVC-Intra 100M 720p25/50 */
		mkTag('a', 'i', '1', '2'): utils.AVCodecIdH264, /* AVC-Intra 100M 1080p25/50 */
		mkTag('a', 'i', '1', '3'): utils.AVCodecIdH264, /* AVC-Intra 100M 1080p24/30/60 */
		mkTag('a', 'i', '1', '5'): utils.AVCodecIdH264, /* AVC-Intra 100M 1080i50 */
		mkTag('a', 'i', '1', '6'): utils.AVCodecIdH264, /* AVC-Intra 100M 1080i60 */
		mkTag('A', 'V', 'i', 'n'): utils.AVCodecIdH264, /* AVC-Intra with implicit SPS/PPS */
		mkTag('a', 'i', 'v', 'x'): utils.AVCodecIdH264, /* XAVC 10-bit 4:2:2 */
		mkTag('r', 'v', '6', '4'): utils.AVCodecIdH264, /* X-Com Radvision */
		mkTag('x', 'a', 'l', 'g'): utils.AVCodecIdH264, /* XAVC-L HD422 produced by FCP */
		mkTag('a', 'v', 'l', 'g'): utils.AVCodecIdH264, /* Panasonic P2 AVC-LongG */
		mkTag('d', 'v', 'a', '1'): utils.AVCodecIdH264, /* AVC-based Dolby Vision derived from avc1 */
		mkTag('d', 'v', 'a', 'v'): utils.AVCodecIdH264, /* AVC-based Dolby Vision derived from avc3 */

		mkTag('v', 'p', '0', '8'): utils.AVCodecIdVP8, /* VP8 */
		mkTag('v', 'p', '0', '9'): utils.AVCodecIdVP9, /* VP9 */
		mkTag('a', 'v', '0', '1'): utils.AVCodecIdAV1, /* AV1 */

		mkTag('m', '1', 'v', ' '): utils.AVCodecIdMPEG1VIDEO,
		mkTag('m', '1', 'v', '1'): utils.AVCodecIdMPEG1VIDEO, /* Apple MPEG-1 Camcorder */
		mkTag('m', 'p', 'e', 'g'): utils.AVCodecIdMPEG1VIDEO, /* MPEG */
		mkTag('m', 'p', '1', 'v'): utils.AVCodecIdMPEG1VIDEO, /* CoreMedia CMVideoCodecType */
		mkTag('m', '2', 'v', '1'): utils.AVCodecIdMPEG2VIDEO, /* Apple MPEG-2 Camcorder */
		mkTag('h', 'd', 'v', '1'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 720p30 */
		mkTag('h', 'd', 'v', '2'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 1080i60 */
		mkTag('h', 'd', 'v', '3'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 1080i50 */
		mkTag('h', 'd', 'v', '4'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 720p24 */
		mkTag('h', 'd', 'v', '5'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 720p25 */
		mkTag('h', 'd', 'v', '6'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 1080p24 */
		mkTag('h', 'd', 'v', '7'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 1080p25 */
		mkTag('h', 'd', 'v', '8'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 1080p30 */
		mkTag('h', 'd', 'v', '9'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 720p60 JVC */
		mkTag('h', 'd', 'v', 'a'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 HDV 720p50 */
		mkTag('m', 'x', '5', 'n'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX NTSC 525/60 50mb/s produced by FCP */
		mkTag('m', 'x', '5', 'p'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX PAL 625/50 50mb/s produced by FCP */
		mkTag('m', 'x', '4', 'n'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX NTSC 525/60 40mb/s produced by FCP */
		mkTag('m', 'x', '4', 'p'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX PAL 625/50 40mb/s produced by FCP */
		mkTag('m', 'x', '3', 'n'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX NTSC 525/60 30mb/s produced by FCP */
		mkTag('m', 'x', '3', 'p'): utils.AVCodecIdMPEG2VIDEO, /* MPEG-2 IMX PAL 625/50 30mb/s produced by FCP */
		mkTag('x', 'd', '5', '1'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 720p30 CBR */
		mkTag('x', 'd', '5', '4'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 720p24 CBR */
		mkTag('x', 'd', '5', '5'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 720p25 CBR */
		mkTag('x', 'd', '5', '9'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 720p60 CBR */
		mkTag('x', 'd', '5', 'a'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 720p50 CBR */
		mkTag('x', 'd', '5', 'b'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 1080i60 CBR */
		mkTag('x', 'd', '5', 'c'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 1080i50 CBR */
		mkTag('x', 'd', '5', 'd'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 1080p24 CBR */
		mkTag('x', 'd', '5', 'e'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 1080p25 CBR */
		mkTag('x', 'd', '5', 'f'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 1080p30 CBR */
		mkTag('x', 'd', 'v', '1'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 720p30 VBR */
		mkTag('x', 'd', 'v', '2'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 1080i60 */
		mkTag('x', 'd', 'v', '3'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 1080i50 VBR */
		mkTag('x', 'd', 'v', '4'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 720p24 VBR */
		mkTag('x', 'd', 'v', '5'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 720p25 VBR */
		mkTag('x', 'd', 'v', '6'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 1080p24 VBR */
		mkTag('x', 'd', 'v', '7'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 1080p25 VBR */
		mkTag('x', 'd', 'v', '8'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 1080p30 VBR */
		mkTag('x', 'd', 'v', '9'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 720p60 VBR */
		mkTag('x', 'd', 'v', 'a'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 720p50 VBR */
		mkTag('x', 'd', 'v', 'b'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 1080i60 VBR */
		mkTag('x', 'd', 'v', 'c'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 1080i50 VBR */
		mkTag('x', 'd', 'v', 'd'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 1080p24 VBR */
		mkTag('x', 'd', 'v', 'e'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 1080p25 VBR */
		mkTag('x', 'd', 'v', 'f'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM EX 1080p30 VBR */
		mkTag('x', 'd', 'h', 'd'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD 540p */
		mkTag('x', 'd', 'h', '2'): utils.AVCodecIdMPEG2VIDEO, /* XDCAM HD422 540p */
		mkTag('A', 'V', 'm', 'p'): utils.AVCodecIdMPEG2VIDEO, /* AVID IMX PAL */
		mkTag('m', 'p', '2', 'v'): utils.AVCodecIdMPEG2VIDEO, /* FCP5 */

		mkTag('m', 'j', 'p', '2'): utils.AVCodecIdJPEG2000, /* JPEG 2000 produced by FCP */

		mkTag('t', 'g', 'a', ' '): utils.AVCodecIdTARGA, /* Truevision Targa */
		mkTag('t', 'i', 'f', 'f'): utils.AVCodecIdTIFF,  /* TIFF embedded in MOV */
		mkTag('g', 'i', 'f', ' '): utils.AVCodecIdGIF,   /* embedded gif files as frames (usually one "click to play movie" frame) */
		mkTag('p', 'n', 'g', ' '): utils.AVCodecIdPNG,
		mkTag('M', 'N', 'G', ' '): utils.AVCodecIdPNG,

		mkTag('v', 'c', '-', '1'): utils.AVCodecIdVC1, /* SMPTE RP 2025 */
		mkTag('a', 'v', 's', '2'): utils.AVCodecIdCAVS,

		mkTag('d', 'r', 'a', 'c'): utils.AVCodecIdDIRAC,
		mkTag('A', 'V', 'd', 'n'): utils.AVCodecIdDNXHD, /* AVID DNxHD */
		mkTag('A', 'V', 'd', 'h'): utils.AVCodecIdDNXHD, /* AVID DNxHR */
		mkTag('H', '2', '6', '3'): utils.AVCodecIdH263,
		mkTag('3', 'I', 'V', 'D'): utils.AVCodecIdMSMPEG4V3, /* 3ivx DivX Doctor */
		mkTag('A', 'V', '1', 'x'): utils.AVCodecIdRAWVIDEO,  /* AVID 1:1x */
		mkTag('A', 'V', 'u', 'p'): utils.AVCodecIdRAWVIDEO,
		mkTag('s', 'g', 'i', ' '): utils.AVCodecIdSGI, /* SGI  */
		mkTag('d', 'p', 'x', ' '): utils.AVCodecIdDPX, /* DPX */
		mkTag('e', 'x', 'r', ' '): utils.AVCodecIdEXR, /* OpenEXR */

		mkTag('a', 'p', 'c', 'h'): utils.AVCodecIdPRORES, /* Apple ProRes 422 High Quality */
		mkTag('a', 'p', 'c', 'n'): utils.AVCodecIdPRORES, /* Apple ProRes 422 Standard Definition */
		mkTag('a', 'p', 'c', 's'): utils.AVCodecIdPRORES, /* Apple ProRes 422 LT */
		mkTag('a', 'p', 'c', 'o'): utils.AVCodecIdPRORES, /* Apple ProRes 422 Proxy */
		mkTag('a', 'p', '4', 'h'): utils.AVCodecIdPRORES, /* Apple ProRes 4444 */
		mkTag('a', 'p', '4', 'x'): utils.AVCodecIdPRORES, /* Apple ProRes 4444 XQ */
		mkTag('f', 'l', 'i', 'c'): utils.AVCodecIdFLIC,

		mkTag('i', 'c', 'o', 'd'): utils.AVCodecIdAIC,

		mkTag('H', 'a', 'p', '1'): utils.AVCodecIdHAP,
		mkTag('H', 'a', 'p', '5'): utils.AVCodecIdHAP,
		mkTag('H', 'a', 'p', 'Y'): utils.AVCodecIdHAP,
		mkTag('H', 'a', 'p', 'A'): utils.AVCodecIdHAP,
		mkTag('H', 'a', 'p', 'M'): utils.AVCodecIdHAP,

		mkTag('D', 'X', 'D', '3'): utils.AVCodecIdDXV,
		mkTag('D', 'X', 'D', 'I'): utils.AVCodecIdDXV,

		mkTag('M', '0', 'R', '0'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '0', 'R', 'A'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '0', 'R', 'G'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '0', 'Y', '0'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '0', 'Y', '2'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '0', 'Y', '4'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'R', 'G'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'R', 'A'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'G', '0'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'Y', '0'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'Y', '2'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'Y', '4'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '8', 'Y', 'A'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '2', 'R', 'A'): utils.AVCodecIdMAGICYUV,
		mkTag('M', '2', 'R', 'G'): utils.AVCodecIdMAGICYUV,

		mkTag('S', 'h', 'r', '0'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '1'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '2'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '3'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '4'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '5'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '6'): utils.AVCodecIdSHEERVIDEO,
		mkTag('S', 'h', 'r', '7'): utils.AVCodecIdSHEERVIDEO,

		mkTag('p', 'x', 'l', 't'): utils.AVCodecIdPIXLET,

		mkTag('n', 'c', 'l', 'c'): utils.AVCodecIdNOTCHLC,

		mkTag('B', 'G', 'G', 'R'): utils.AVCodecIdRAWVIDEO, /* ASC Bayer BGGR */

	}

	audioTags = map[uint32]utils.AVCodecID{
		mkTag('m', 'p', '4', 'a'): utils.AVCodecIdAAC,
		mkTag('a', 'c', '-', '3'): utils.AVCodecIdAC3, /* ETSI TS 102 366 Annex F */
		mkTag('s', 'a', 'c', '3'): utils.AVCodecIdAC3, /* Nero Recode */
		mkTag('i', 'm', 'a', '4'): utils.AVCodecIdADPCMIMAQT,
		mkTag('a', 'l', 'a', 'c'): utils.AVCodecIdALAC,
		mkTag('s', 'a', 'm', 'r'): utils.AVCodecIdAMRNB, /* AMR-NB 3gp */
		mkTag('s', 'a', 'w', 'b'): utils.AVCodecIdAMRWB, /* AMR-WB 3gp */
		mkTag('d', 't', 's', 'c'): utils.AVCodecIdDTS,   /* DTS formats prior to DTS-HD */
		mkTag('d', 't', 's', 'h'): utils.AVCodecIdDTS,   /* DTS-HD audio formats */
		mkTag('d', 't', 's', 'l'): utils.AVCodecIdDTS,   /* DTS-HD Lossless formats */
		mkTag('d', 't', 's', 'e'): utils.AVCodecIdDTS,   /* DTS Express */
		mkTag('D', 'T', 'S', ' '): utils.AVCodecIdDTS,   /* non-standard */
		mkTag('e', 'c', '-', '3'): utils.AVCodecIdEAC3,  /* ETSI TS 102 366 Annex F (only valid in ISOBMFF) */
		mkTag('v', 'd', 'v', 'a'): utils.AVCodecIdDVAUDIO,
		mkTag('d', 'v', 'c', 'a'): utils.AVCodecIdDVAUDIO,
		mkTag('a', 'g', 's', 'm'): utils.AVCodecIdGSM,
		mkTag('i', 'l', 'b', 'c'): utils.AVCodecIdILBC,
		mkTag('M', 'A', 'C', '3'): utils.AVCodecIdMACE3,
		mkTag('M', 'A', 'C', '6'): utils.AVCodecIdMACE6,
		mkTag('.', 'm', 'p', '1'): utils.AVCodecIdMP1,
		mkTag('.', 'm', 'p', '2'): utils.AVCodecIdMP2,
		mkTag('.', 'm', 'p', '3'): utils.AVCodecIdMP3,
		mkTag('m', 'p', '3', ' '): utils.AVCodecIdMP3, /* vlc */
		mkTag('t', 'e', 's', 't'): utils.AVCodecIdMP3,
		mkTag('n', 'm', 'o', 's'): utils.AVCodecIdNELLYMOSER, /* Flash Media Server */
		mkTag('N', 'E', 'L', 'L'): utils.AVCodecIdNELLYMOSER, /* Perian */
		mkTag('a', 'l', 'a', 'w'): utils.AVCodecIdPCMALAW,
		mkTag('f', 'l', '3', '2'): utils.AVCodecIdPCMF32BE,
		mkTag('f', 'l', '3', '2'): utils.AVCodecIdPCMF32LE,
		mkTag('f', 'l', '6', '4'): utils.AVCodecIdPCMF64BE,
		mkTag('f', 'l', '6', '4'): utils.AVCodecIdPCMF64LE,
		mkTag('u', 'l', 'a', 'w'): utils.AVCodecIdPCMMULAW,
		mkTag('t', 'w', 'o', 's'): utils.AVCodecIdPCMS16BE,
		mkTag('s', 'o', 'w', 't'): utils.AVCodecIdPCMS16LE,
		mkTag('l', 'p', 'c', 'm'): utils.AVCodecIdPCMS16BE,
		mkTag('l', 'p', 'c', 'm'): utils.AVCodecIdPCMS16LE,
		mkTag('i', 'n', '2', '4'): utils.AVCodecIdPCMS24BE,
		mkTag('i', 'n', '2', '4'): utils.AVCodecIdPCMS24LE,
		mkTag('i', 'n', '3', '2'): utils.AVCodecIdPCMS32BE,
		mkTag('i', 'n', '3', '2'): utils.AVCodecIdPCMS32LE,
		mkTag('s', 'o', 'w', 't'): utils.AVCodecIdPCMS8,
		mkTag('r', 'a', 'w', ' '): utils.AVCodecIdPCMU8,
		mkTag('N', 'O', 'N', 'E'): utils.AVCodecIdPCMU8,
		mkTag('Q', 'c', 'l', 'p'): utils.AVCodecIdQCELP,
		mkTag('Q', 'c', 'l', 'q'): utils.AVCodecIdQCELP,
		mkTag('s', 'q', 'c', 'p'): utils.AVCodecIdQCELP, /* ISO Media fourcc */
		mkTag('Q', 'D', 'M', '2'): utils.AVCodecIdQDM2,
		mkTag('Q', 'D', 'M', 'C'): utils.AVCodecIdQDMC,
		mkTag('s', 'p', 'e', 'x'): utils.AVCodecIdSPEEX,        /* Flash Media Server */
		mkTag('S', 'P', 'X', 'N'): utils.AVCodecIdSPEEX,        /* ZygoAudio (quality 10 mode) */
		mkTag('s', 'e', 'v', 'c'): utils.AVCodecIdEVRC,         /* 3GPP2 */
		mkTag('s', 's', 'm', 'v'): utils.AVCodecIdSMV,          /* 3GPP2 */
		mkTag('f', 'L', 'a', 'C'): utils.AVCodecIdFLAC,         /* nonstandard */
		mkTag('m', 'l', 'p', 'a'): utils.AVCodecIdTRUEHD,       /* mp4ra.org */
		mkTag('O', 'p', 'u', 's'): utils.AVCodecIdOPUS,         /* mp4ra.org */
		mkTag('m', 'h', 'm', '1'): utils.AVCodecIdMPEGH3DAUDIO, /* MPEG-H 3D Audio bitstream */
	}

	subtitleTags = map[uint32]utils.AVCodecID{
		mkTag('t', 'e', 'x', 't'): utils.AVCodecIdMOVTEXT,
		mkTag('t', 'x', '3', 'g'): utils.AVCodecIdMOVTEXT,
		mkTag('c', '6', '0', '8'): utils.AVCodecIdEIA608,
	}

	dataTags = map[uint32]utils.AVCodecID{
		mkTag('g', 'p', 'm', 'd'): utils.AVCodecIdBINDATA,
	}

	println(videoTags)
	println(audioTags)
	println(subtitleTags)
	println(dataTags)
}
