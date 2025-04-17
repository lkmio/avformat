package bufio

import (
	"fmt"
	"testing"
)

func TestBitsReader(t *testing.T) {
	data := []byte{0b10101010, 0b11001100, 0b11110000} // 示例数据
	startBit := 4                                      // 从第4个bit开始读取
	bitLength := 8                                     // 读取8个bit

	reader := BitsReader{Data: data, Offset: startBit}
	value := reader.Read(bitLength)
	fmt.Printf("读取的值: %b\n", value) // 输出二进制格式
}
