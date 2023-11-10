package librtmp

import (
	"encoding/binary"
	"math/rand"
	"time"
)

const (
	VERSION             = 3
	HandshakePacketSize = 1536
	DefaultPort         = 1935
	RTMPSDefaultPort    = 443
)

/**
0 1 2 3 4 5 6 7
+-+-+-+-+-+-+-+-+
| version |
+-+-+-+-+-+-+-+-+
C0 and S0 bits
*/

/**
 The C1 and S1 packets are 1536 octets long, consisting of the
 following fields:
0 1 2 3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| time (4 bytes) |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| zero (4 bytes) |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| random bytes |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| random bytes |
| (cont) |
| .... |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
C1 and S1 bits
*/

/**
0 1 2 3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 | time (4 bytes) |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 | time2 (4 bytes) |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 | random echo |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 | random echo |
 | (cont) |
 | .... |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 C2 and S2 bits
*/

// GenerateC0C1 C0+C1一起发，收到S1发C2。
func GenerateC0C1(dst []byte) int {
	size := 1 + HandshakePacketSize

	dst[0] = VERSION
	//ffmpeg后面写flash client version 有的写C1。
	//gen random bytes
	for i := 9; i < size; i++ {
		dst[i] = byte(rand.Intn(255))
	}

	return size
}

// GenerateS0S1S2 Server将S0S1S2一起发送
func GenerateS0S1S2(dst []byte, c1 []byte) int {
	size := 1 + HandshakePacketSize*2
	//S0
	dst[0] = VERSION
	//S1
	time1 := uint32(time.Now().Second())
	binary.BigEndian.PutUint32(dst[1:], time1)
	binary.BigEndian.PutUint32(dst[5:], 0)
	randEcho(dst[9:], time1)
	//S2
	time2 := uint32(time.Now().Second())
	copy(dst[1+HandshakePacketSize:], c1)
	binary.BigEndian.PutUint32(dst[1+HandshakePacketSize:], time2)

	return size
}

func randEcho(dst []byte, time uint32) {
	rand.Seed(int64(time))
	end := HandshakePacketSize - 8
	for i := 0; i < end; i++ {
		dst[i] = byte(rand.Intn(255))
	}
}
