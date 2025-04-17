package avc

const (
	H264NalUnspecified         = 0
	H264NalSlice               = 1
	H264NalDpa                 = 2
	H264NalDpb                 = 3
	H264NalDPC                 = 4
	H264NalIDRSlice            = 5
	H264NalSEI                 = 6
	H264NalSPS                 = 7
	H264NalPPS                 = 8
	H264NalAUD                 = 9
	H264NalEndSequence         = 10
	H264NalEndStream           = 11
	H264NalFillerData          = 12
	H264NalSpsExt              = 13
	H264NalPREFIX              = 14
	H264NalSubSps              = 15
	H264NalDPS                 = 16
	H264NalRESERVED17          = 17
	H264NalRESERVED18          = 18
	H264NalAuxiliarySlice      = 19
	H264NalExtensionSlice      = 20
	H264NalDepthExtensionSlice = 21
	H264NalRESERVED22          = 22
	H264NalRESERVED23          = 23
	H264NalUNSPECIFIED24       = 24 //24-31 不保留RTP打包会用到
	H264NalUNSPECIFIED25       = 25
	H264NalUNSPECIFIED26       = 26
	H264NalUNSPECIFIED27       = 27
	H264NalUNSPECIFIED28       = 28
	H264NalUNSPECIFIED29       = 29
	H264NalUNSPECIFIED30       = 30
	H264NalUNSPECIFIED31       = 31
)

const (
	// H264MaxSpsCount 7.4.2.1.1: seq_parameter_set_id is in [0, 31].
	H264MaxSpsCount = 32
	// H264MaxPpsCount 7.4.2.2: pic_parameter_set_id is in [0, 255].
	H264MaxPpsCount = 256

	// H264MaxDpbFrames A.3: MaxDpbFrames is bounded above by 16.
	H264MaxDpbFrames = 16
	// H264MaxRefs 7.4.2.1.1: max_num_ref_frames is in [0, MaxDpbFrames], and
	// each reference frame can have two fields.
	H264MaxRefs = 2 * H264MaxDpbFrames

	// H264MaxRPLMCount 7.4.3.1: modification_of_pic_nums_idc is not equal to 3 at most
	// num_ref_idx_lN_active_minus1 + 1 times (that is, once for each
	// possible reference), then equal to 3 once.
	H264MaxRPLMCount = H264MaxRefs + 1

	// H264MaxMMCOCount 7.4.3.3: in the worst case, we begin with a full short-term
	// reference picture list.  Each picture in turn is moved to the
	// long-term list (type 3) and then discarded from there (type 2).
	// Then, we set the length of the long-term list (type 4), mark
	// the current picture as long-term (type 6) and terminate the
	// process (type 0).
	H264MaxMMCOCount = H264MaxRefs*2 + 3

	// H264MaxSliceGroups A.2.1, A.2.3: profiles supporting FMO constrain
	// num_slice_groups_minus1 to be in [0, 7].
	H264MaxSliceGroups = 8

	// H264MaxCpbCnt E.2.2: cpb_cnt_minus1 is in [0, 31].
	H264MaxCpbCnt = 32

	// H264MaxMbPicSize A.3: in table A-1 the highest level allows a MaxFS of 139264.
	H264MaxMbPicSize = 139264
	// H264MaxMbWidth A.3.1, A.3.2: PicWidthInMbs and PicHeightInMbs are constrained
	// to be not greater than sqrt(MaxFS * 8).  Hence height/width are
	// bounded above by sqrt(139264 * 8) = 1055.5 macroblocks.
	H264MaxMbWidth  = 1055
	H264MaxMbHeight = 1055
	H264MaxWidth    = H264MaxMbWidth * 16
	H264MaxHeight   = H264MaxMbHeight * 16
)
