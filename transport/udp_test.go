package transport

import (
	"net"
	"testing"
)

type UDPHandler struct {
}

func (U *UDPHandler) OnConnected(conn net.Conn) {
}

func (U *UDPHandler) OnPacket(conn net.Conn, data []byte) {
	conn.Write(data)
}

func (U *UDPHandler) OnDisConnected(conn net.Conn, err error) {
}

func TestUDPServer(t *testing.T) {
	transport := UDPTransport{}
	handler := &UDPHandler{}
	transport.SetHandler(handler)

	addr := "0.0.0.0:20000"
	if err := transport.Bind(addr); err != nil {
		panic(err)
	}

	println("启动UDPServer成功 addr:" + addr)

	select {}
}
