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
	println("udp 断开链接")
}

func TestUDPServer(t *testing.T) {
	transport := UDPServer{}
	handler := &UDPHandler{}
	transport.SetHandler(handler)

	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:20000")
	if err := transport.Bind(addr); err != nil {
		panic(err)
	}

	println("启动UDPServer成功 addr:" + addr.String())

	select {}
}
