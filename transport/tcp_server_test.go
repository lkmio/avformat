package transport

import (
	"net"
	"testing"
)

type TCPServerHandler struct {
}

func (T *TCPServerHandler) OnConnected(conn net.Conn) {
	println("客户端连接: " + conn.RemoteAddr().String())
}

func (T *TCPServerHandler) OnPacket(conn net.Conn, data []byte) {
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}
}

func (T *TCPServerHandler) OnDisConnected(conn net.Conn, err error) {
	println("客户端断开连接: " + conn.RemoteAddr().String())
}

func TestTCPServer(t *testing.T) {
	server := TCPServer{}
	handler := &TCPServerHandler{}
	server.SetHandler(handler)
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:8000")

	if err := server.Bind(addr); err != nil {
		panic(err)
	}

	println("成功监听:" + addr.String())
	select {}
}
