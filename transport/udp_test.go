package transport

import (
	"net"
	"runtime"
	"testing"
)

type UDPHandler struct {
}

func (U *UDPHandler) OnConnected(conn net.Conn) []byte {
	return nil
}

func (U *UDPHandler) OnPacket(conn net.Conn, data []byte) []byte {
	conn.Write(data)
	return nil
}

func (U *UDPHandler) OnDisConnected(conn net.Conn, err error) {
	println("udp 断开链接")
}

func TestUDPServer(t *testing.T) {
	udpServer := UDPServer{
		ReuseServer: ReuseServer{
			EnableReuse:      true,
			ConcurrentNumber: runtime.NumCPU(),
		},
	}

	udpServer.SetHandler(&UDPHandler{})
	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:20000")
	if err := udpServer.Bind(addr); err != nil {
		panic(err)
	}

	println("启动UDPServer成功 addr:" + addr.String())
	udpServer.Receive()

	select {}
}
