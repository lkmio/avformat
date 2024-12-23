package libflv

import (
	"encoding/hex"
	"github.com/lkmio/avformat/utils"
	"testing"
)

func TestAMFO(t *testing.T) {
	strings := []string{""}

	for _, str := range strings {
		bytes, err := hex.DecodeString(str)
		utils.Assert(err == nil)

		amf0 := AMF0{}
		err = amf0.Unmarshal(bytes)
		utils.Assert(err == nil)

		dst := make([]byte, len(bytes))
		n, err := amf0.Marshal(dst)
		toString := hex.EncodeToString(dst[:n])
		println(toString)
		utils.Assert(err == nil)
		utils.Assert(n == len(bytes))

	}
}
