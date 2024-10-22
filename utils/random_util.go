package utils

import (
	"math/rand"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

// RandomIntInRange 返回在[min, max]范围内的随机整数
func RandomIntInRange(min, max int) int {
	if min > max {
		panic("min should not be greater than max")
	}

	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}
