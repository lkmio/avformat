package transport

import (
	"context"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"syscall"
	"time"
)

type TCPServer struct {
	transport
	listener net.Listener
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
		t.listener = listen
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

		go recvTcp(t.ctx, tcp, t.handler)
	}
}

func (t *TCPServer) Close() {
	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}

	t.transport.Close()
}

func recvTcp(ctx context.Context, conn net.Conn, handler Handler) {
	extraConn := &Conn{conn: conn, buffer: nil}
	if handler != nil {
		bytes := handler.OnConnected(extraConn)
		if bytes == nil {
			bytes = make([]byte, DefaultTCPRecvBufferSize)
		}

		extraConn.buffer = bytes
	}

	var n int
	var err error
	var receiveBuffer []byte
	for ctx.Err() == nil {
		if receiveBuffer == nil {
			receiveBuffer = extraConn.buffer
		}

		n, err = conn.Read(receiveBuffer)
		if err != nil {
			break
		}

		if n > 0 && handler != nil {
			receiveBuffer = handler.OnPacket(extraConn, receiveBuffer[:n])
		}
	}

	_ = conn.Close()

	if handler != nil {
		handler.OnDisConnected(extraConn, err)
	}
}
