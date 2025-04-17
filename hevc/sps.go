package hevc

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/lkmio/avformat/avc"
	"github.com/lkmio/avformat/bufio"
)

type HEVCSPSInfo struct {
	//avc.SPS

	numTemporalLayers                uint
	temporalIdNested                 uint
	chromaFormat                     uint
	PicWidthInLumaSamples            uint
	PicHeightInLumaSamples           uint
	bitDepthLumaMinus8               uint
	bitDepthChromaMinus8             uint
	generalProfileSpace              uint
	generalTierFlag                  uint
	generalProfileIDC                uint
	generalProfileCompatibilityFlags uint32
	generalConstraintIndicatorFlags  uint64
	generalLevelIDC                  uint
	fps                              uint
	Width                            int
	Height                           int
}

func ParseSPS(sps []byte) (ctx HEVCSPSInfo, err error) {
	sps = avc.RemoveStartCode(sps)

	if len(sps) < 2 {
		err = errors.New("incorrect Unit Size")
		return
	}

	rbsp := nal2rbsp(sps[2:])
	br := &bufio.GolombBitReader{R: bytes.NewReader(rbsp)}
	if _, err = br.ReadBits(4); err != nil {
		return
	}
	spsMaxSubLayersMinus1, err := br.ReadBits(3)
	if err != nil {
		return
	}

	if spsMaxSubLayersMinus1+1 > ctx.numTemporalLayers {
		ctx.numTemporalLayers = spsMaxSubLayersMinus1 + 1
	}
	if ctx.temporalIdNested, err = br.ReadBit(); err != nil {
		return
	}
	if err = parsePTL(br, &ctx, spsMaxSubLayersMinus1); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	var cf uint
	if cf, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	ctx.chromaFormat = uint(cf)
	if ctx.chromaFormat == 3 {
		if _, err = br.ReadBit(); err != nil {
			return
		}
	}
	if ctx.PicWidthInLumaSamples, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	ctx.Width = int(ctx.PicWidthInLumaSamples)
	if ctx.PicHeightInLumaSamples, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	ctx.Height = int(ctx.PicHeightInLumaSamples)
	conformanceWindowFlag, err := br.ReadBit()
	if err != nil {
		return
	}
	if conformanceWindowFlag != 0 {
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
	}

	var bdlm8 uint
	if bdlm8, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	ctx.bitDepthChromaMinus8 = uint(bdlm8)
	var bdcm8 uint
	if bdcm8, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	ctx.bitDepthChromaMinus8 = uint(bdcm8)

	_, err = br.ReadExponentialGolombCode()
	if err != nil {
		return
	}
	spsSubLayerOrderingInfoPresentFlag, err := br.ReadBit()
	if err != nil {
		return
	}
	var i uint
	if spsSubLayerOrderingInfoPresentFlag != 0 {
		i = 0
	} else {
		i = spsMaxSubLayersMinus1
	}
	for ; i <= spsMaxSubLayersMinus1; i++ {
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
		if _, err = br.ReadExponentialGolombCode(); err != nil {
			return
		}
	}

	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	if _, err = br.ReadExponentialGolombCode(); err != nil {
		return
	}
	return
}

func parsePTL(br *bufio.GolombBitReader, ctx *HEVCSPSInfo, maxSubLayersMinus1 uint) error {
	var err error
	var ptl HEVCSPSInfo
	if ptl.generalProfileSpace, err = br.ReadBits(2); err != nil {
		return err
	}
	if ptl.generalTierFlag, err = br.ReadBit(); err != nil {
		return err
	}
	if ptl.generalProfileIDC, err = br.ReadBits(5); err != nil {
		return err
	}
	if ptl.generalProfileCompatibilityFlags, err = br.ReadBits32(32); err != nil {
		return err
	}
	if ptl.generalConstraintIndicatorFlags, err = br.ReadBits64(48); err != nil {
		return err
	}
	if ptl.generalLevelIDC, err = br.ReadBits(8); err != nil {
		return err
	}
	updatePTL(ctx, &ptl)
	if maxSubLayersMinus1 == 0 {
		return nil
	}
	subLayerProfilePresentFlag := make([]uint, maxSubLayersMinus1)
	subLayerLevelPresentFlag := make([]uint, maxSubLayersMinus1)
	for i := uint(0); i < maxSubLayersMinus1; i++ {
		if subLayerProfilePresentFlag[i], err = br.ReadBit(); err != nil {
			return err
		}
		if subLayerLevelPresentFlag[i], err = br.ReadBit(); err != nil {
			return err
		}
	}
	if maxSubLayersMinus1 > 0 {
		for i := maxSubLayersMinus1; i < 8; i++ {
			if _, err = br.ReadBits(2); err != nil {
				return err
			}
		}
	}
	for i := uint(0); i < maxSubLayersMinus1; i++ {
		if subLayerProfilePresentFlag[i] != 0 {
			if _, err = br.ReadBits32(32); err != nil {
				return err
			}
			if _, err = br.ReadBits32(32); err != nil {
				return err
			}
			if _, err = br.ReadBits32(24); err != nil {
				return err
			}
		}

		if subLayerLevelPresentFlag[i] != 0 {
			if _, err = br.ReadBits(8); err != nil {
				return err
			}
		}
	}
	return nil
}

func updatePTL(ctx, ptl *HEVCSPSInfo) {
	ctx.generalProfileSpace = ptl.generalProfileSpace

	if ptl.generalTierFlag > ctx.generalTierFlag {
		ctx.generalLevelIDC = ptl.generalLevelIDC

		ctx.generalTierFlag = ptl.generalTierFlag
	} else {
		if ptl.generalLevelIDC > ctx.generalLevelIDC {
			ctx.generalLevelIDC = ptl.generalLevelIDC
		}
	}

	if ptl.generalProfileIDC > ctx.generalProfileIDC {
		ctx.generalProfileIDC = ptl.generalProfileIDC
	}

	ctx.generalProfileCompatibilityFlags &= ptl.generalProfileCompatibilityFlags

	ctx.generalConstraintIndicatorFlags &= ptl.generalConstraintIndicatorFlags
}

func nal2rbsp(nal []byte) []byte {
	return bytes.Replace(nal, []byte{0x0, 0x0, 0x3}, []byte{0x0, 0x0}, -1)
}

func NewCodecDataFromHEVCDecoderConfigurationRecord(record []byte) (*HEVCDecoderConfigurationRecord, *HEVCSPSInfo, error) {
	confRecord := HEVCDecoderConfigurationRecord{}
	if err := confRecord.Unmarshal(record); err != nil {
		return nil, nil, err
	}
	spsInfo, err := ParseSPS(confRecord.SPSList[0])
	if err != nil {
		return nil, nil, fmt.Errorf("h265parser: parse SPS failed(%s)", err)
	}
	return &confRecord, &spsInfo, nil
}
