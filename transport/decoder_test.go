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

func TestNewDelimiterFrameDecoder(t *testing.T) {
	data := "123456abc789abchello worldabctest"
	decoder := NewDelimiterFrameDecoder(1024*1024*2, []byte("abc"), func(bytes []byte) {
		println(string(bytes))
	})

	err := decoder.Input([]byte(data))
	if err != nil {
		panic(err)
	}

}
