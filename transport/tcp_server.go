package transport

import (
	"context"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"syscall"
	"time"
)

type TCPServer struct {
	transportImpl
}

func (t *TCPServer) Bind(addr net.Addr) error {
	utils.Assert(t.handler != nil)

	config := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				//syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				//syscall.SO_REUSEADDR
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
		},
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())

	if listen, err := config.Listen(t.ctx, "tcp", addr.String()); err != nil {
		return err
	} else {
		time.Sleep(100 * time.Millisecond)
		go t.accept(listen.(*net.TCPListener))
		return nil
	}
}

func (t *TCPServer) accept(listener *net.TCPListener) {
	for t.ctx.Err() == nil {
		tcp, err := listener.AcceptTCP()
		if err != nil {
			println(err.Error())
			continue
		}

		go recv(t.ctx, tcp, t.handler)
	}
}

func recv(ctx context.Context, conn net.Conn, handler Handler) {
	extraConn := &Conn{conn: conn, buffer: make([]byte, DefaultTCPRecvBufferSize)}
	if handler != nil {
		handler.OnConnected(extraConn)
	}

	var n int
	var err error
	for ctx.Err() == nil {
		n, err = conn.Read(extraConn.buffer)
		if err != nil {
			break
		}

		if n > 0 && handler != nil {
			handler.OnPacket(extraConn, extraConn.buffer[:n])
		}
	}

	_ = conn.Close()

	if handler != nil {
		handler.OnDisConnected(extraConn, err)
	}
}
