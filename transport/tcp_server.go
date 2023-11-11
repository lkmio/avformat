package transport

import (
	"context"
	"net"
	"syscall"
	"time"
)

const (
	DefaultTCPRecvBufferSize = 4096
)

// Handler 传输事件处理器，负责连接传输和断开连接的回调
type Handler interface {
	OnConnected(conn net.Conn)

	OnPacket(conn net.Conn, data []byte)

	OnDisConnected(conn net.Conn, err error)
}

type ITransport interface {
	Bind(addr string) error

	Close()

	SetHandler(handler Handler)
}

type transportImpl struct {
	handler Handler

	ctx    context.Context
	cancel context.CancelFunc
}

func (impl *transportImpl) SetHandler(handler Handler) {
	impl.handler = handler
}

func (impl *transportImpl) Close() {
	impl.cancel()
}

type TCPServer struct {
	transportImpl
}

func (t *TCPServer) Bind(addr string) error {
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

	if listen, err := config.Listen(t.ctx, "tcp", addr); err != nil {
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
