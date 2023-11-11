package avformat

import "testing"

func TestAssert(t *testing.T) {

	//Assert(true)
	//Assert(false)

	data1 := []byte("Hello, ")
	data2 := []byte("world!")
	data3 := []byte(" How are you?") // 合并多个切片为一个切片
	combinedData := append(append(data1, data2...), data3...)
	println(combinedData)
}
