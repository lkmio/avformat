package collections

import "testing"

func TestBlockBuffer(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		buffer := NewDirectBlockBuffer(24)
		for i := 0; i < 100; i++ {
			buffer.Write([]byte{byte(i)})
		}

		for i := 0; i < 100; i++ {
			buffer.Pop()
		}
	})

	t.Run("rb", func(t *testing.T) {
		buffer := NewDirectBlockBuffer(24)
		for i := 0; i < 100; i++ {
			buffer.Write([]byte{byte(i)})
		}

		for i := 0; i < 100; i++ {
			buffer.Pop()
		}
	})
}
