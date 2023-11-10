package transport

import (
	"net"
	"testing"
)

type TCPClientHandler struct {
}

func (T *TCPClientHandler) OnConnected(conn net.Conn) {
	println("Client:" + conn.LocalAddr().String() + " 链接成功")
	conn.Write([]byte("hello world!"))

}

func (T *TCPClientHandler) OnPacket(conn net.Conn, data []byte) {

}

func (T *TCPClientHandler) OnDisConnected(conn net.Conn, err error) {
	println("Client:" + conn.LocalAddr().String() + " 断开链接")
}

func TestTCPClient(t *testing.T) {
	var clients []*TCPClient
	handler := &TCPClientHandler{}
	for i := 0; i < 100; i++ {
		client := &TCPClient{}
		go func() {
			serverAddr, _ := net.ResolveTCPAddr("tcp", "192.168.2.145:8000")
			client.SetHandler(handler)

			if err := client.Connect(nil, serverAddr); err != nil {
				panic(err)
			}
		}()

		clients = append(clients, client)
	}

	select {}
}
