package avc

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestAVCDecoderConfigurationRecord(t *testing.T) {
	data := "0142c01effe100186742c01eda01e0089f961000000300100000030320f162ea01000568ce0f2c80"
	extraData, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}

	record := AVCDecoderConfigurationRecord{}
	err = record.Unmarshal(extraData)
	if err != nil {
		panic(err)
	}

	b, err := ExtraDataToAnnexB(extraData)
	if err != nil {
		panic(err)
	}

	println("extra data AnnexB:%s", hex.EncodeToString(b))

	_, sps, err := NewCodecDataFromAVCDecoderConfigurationRecord(extraData)
	if err != nil {
		panic(err)
	}

	println(fmt.Sprintf("width:%d height:%d", sps.Width, sps.Height))
}
