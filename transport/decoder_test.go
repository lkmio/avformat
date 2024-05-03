package transport

import "testing"

func TestLengthFieldFrameDecoder_Input(t *testing.T) {
	decoder := NewLengthFieldFrameDecoder(0xFFFF, 2, func(bytes []byte) {
		println(bytes)
	})

	bytes := [1024]byte{0x00, 0x3, 0x1, 0x2}

	decoder.Input(bytes[:4])
	decoder.Input(bytes[:1])
}
