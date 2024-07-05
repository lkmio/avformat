package transport

import (
	"net"
	"testing"
	"time"
)

func TestUDPClient(t *testing.T) {

	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:20001")
	remoteAddr, _ := net.ResolveUDPAddr("udp", "192.168.31.112:40000")

	client := UDPClient{}
	err := client.Connect(addr, remoteAddr)
	if err != nil {
		panic(err)
	}
	client.SetHandler2(nil, func(conn net.Conn, data []byte) []byte {
		println("recv data:" + string(data))
		return nil
	}, nil)

	go client.Receive()

	for {
		msg := "hello!"
		err := client.Write([]byte(msg))
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second)
	}
}
