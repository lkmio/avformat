package transport

import (
	"fmt"
	"net"
	"testing"
	"time"
)

type TCPClientHandler struct {
}

func (T *TCPClientHandler) OnConnected(conn net.Conn) []byte {
	println("Client:" + conn.LocalAddr().String() + " 链接成功")
	conn.Write([]byte("hello world!"))
	return nil
}

func (T *TCPClientHandler) OnPacket(conn net.Conn, data []byte) []byte {
	return nil
}

func (T *TCPClientHandler) OnDisConnected(conn net.Conn, err error) {
	println("Client:" + conn.LocalAddr().String() + " 断开链接")
}

func TestTCPClient(t *testing.T) {
	d := net.Dialer{
		LocalAddr: &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 9000}, // 尝试绑定到本地的8888端口
		Timeout:   5 * time.Second,
	}

	//net.DialTCP("tcp", &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8888}, &net.TCPAddr{IP: net.ParseIP("192.168.31.112"), Port: 40000})
	conn, err := d.Dial("tcp", "192.168.31.112:40000")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		return
	}
	defer conn.Close()

	select {}

	var clients []*TCPClient

	handler_ := &TCPClientHandler{}

	for i := 0; i < 100; i++ {
		client := &TCPClient{}
		go func() {
			serverAddr, _ := net.ResolveTCPAddr("tcp", "192.168.2.145:8000")
			client.SetHandler(handler_)

			if err := client.Connect(nil, serverAddr); err != nil {
				panic(err)
			}

			client.Receive()
		}()

		clients = append(clients, client)
	}

	select {}
}
