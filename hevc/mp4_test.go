package hevc

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestNameHEVCDecoderConfigurationRecord(t *testing.T) {
	data := "0101600000009000000000005df000fcfdf8f800000f03a00001001840010c01ffff01600000030090000003000003005d999809a10001002d42010101600000030090000003000003005da00280802d165999a4932b9a808080820000030002000003003210a2000100074401c172b46240"
	extraData, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}

	record := HEVCDecoderConfigurationRecord{}
	err = record.Unmarshal(extraData)
	if err != nil {
		panic(err)
	}

	b, err := ExtraDataToAnnexB(extraData)
	if err != nil {
		panic(err)
	}

	println("extra data AnnexB:%s", hex.EncodeToString(b))

	_, sps, err := NewCodecDataFromHEVCDecoderConfigurationRecord(extraData)
	if err != nil {
		panic(err)
	}

	println(fmt.Sprintf("width:%d height:%d", sps.Width, sps.Height))
}
