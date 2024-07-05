package transport

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"testing"
	"time"
)

type TCPServerHandler struct {
}

func (T *TCPServerHandler) OnConnected(conn net.Conn) []byte {
	println("客户端连接: " + conn.RemoteAddr().String())
	return nil
}

func (T *TCPServerHandler) OnPacket(conn net.Conn, data []byte) []byte {
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}
	return nil
}

func (T *TCPServerHandler) OnDisConnected(conn net.Conn, err error) {
	println("客户端断开连接: " + conn.RemoteAddr().String())
}

func TestTCPServer(t *testing.T) {
	server := TCPServer{
		ReuseServer: ReuseServer{
			EnableReuse:      true,
			ConcurrentNumber: runtime.NumCPU(),
		},
	}
	server.SetHandler(&TCPServerHandler{})
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:8000")

	if err := server.Bind(addr); err != nil {
		panic(err)
	}

	server.Accept()
	println("成功监听:" + addr.String())

	loadConfigError := http.ListenAndServe(":20000", nil)
	if loadConfigError != nil {
		panic(loadConfigError)
	}

	timeout, _ := context.WithTimeout(context.Background(), time.Second*1000)
	select {
	case <-timeout.Done():
		break
	}
}
