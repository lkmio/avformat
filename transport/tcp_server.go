package transport

import (
	"context"
	"github.com/lkmio/avformat/utils"
	"net"
)

type TCPServer struct {
	ReuseServer
	listeners []*net.TCPListener
}

func (t *TCPServer) Bind(addr net.Addr) error {
	utils.Assert(t.listeners == nil)

	random := addr == nil
	if random {
		t.ConcurrentNumber = 1
		addr, _ = net.ResolveTCPAddr("tcp", ":0")
	}

	config := net.ListenConfig{
		Control: t.GetSetOptFunc(),
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	for i := 0; i < t.ComputeConcurrentNumber(); i++ {

		listen, err := config.Listen(t.ctx, "tcp", addr.String())
		if err != nil {
			t.Close()
			return err
		}

		t.listeners = append(t.listeners, listen.(*net.TCPListener))

		if random {
			t.setListenAddr(listen.Addr())
		} else {
			t.setListenAddr(addr)
		}
	}

	return nil
}

func (t *TCPServer) Accept() {
	utils.Assert(t.handler != nil)
	utils.Assert(len(t.listeners) > 0)

	for _, listener := range t.listeners {
		go t.doAccept(listener)
	}
}

func (t *TCPServer) doAccept(listener *net.TCPListener) {
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
	for _, listener := range t.listeners {
		listener.Close()
	}

	t.listeners = nil
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
