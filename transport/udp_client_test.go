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

func TestUDPDefaultClient(t *testing.T) {

	tcp, err := net.ListenTCP("tcp", nil)
	a := tcp.Addr()
	println(a)

	udp, err := net.ListenUDP("udp", nil)
	if err != nil {
		panic(err)

	}

	addr := udp.LocalAddr()
	println(addr)

	client := UDPClient{}
	client.SetHandler2(nil, func(conn net.Conn, data []byte) []byte {
		println("recv data:" + string(data))
		return nil
	},
		nil)
	client.Connect(nil, nil)

	go client.Receive()

	remoteAddr, _ := net.ResolveUDPAddr("udp", "192.168.2.148:40000")

	for {
		msg := "hello!"
		err := client.WriteTo([]byte(msg), remoteAddr)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second)
	}
}
