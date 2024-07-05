package transport

import (
	"net"
	"strconv"
	"testing"
)

func TestName(t *testing.T) {
	manager := NewTransportManager(20000, 40000)
	server, err := manager.NewTCPServer("0.0.0.0")
	if err != nil {
		panic(err)
	}

	server.SetHandler2(nil, func(conn net.Conn, data []byte) []byte {
		conn.Write(data)
		return nil
	}, nil)

	server.Accept()

	println("启动tcp server:" + strconv.Itoa(server.listenPort))

	udpServer, err := manager.NewUDPServer("0.0.0.0")
	if err != nil {
		panic(err)
	}

	udpServer.SetHandler2(nil, func(conn net.Conn, data []byte) []byte {
		conn.Write(data)
		return nil
	}, nil)

	udpServer.Receive()

	println("启动udp server:" + strconv.Itoa(udpServer.listenPort))
	select {}
}
