package transport

import (
	"context"
	"net"
)

type TCPClient struct {
	transportImpl
	tcp net.Conn
}

func (t *TCPClient) Bind(addr net.Addr) error {
	panic("please use the connect func")
}

func (t *TCPClient) Connect(local, addr *net.TCPAddr) error {
	dialer := net.Dialer{
		LocalAddr: local,
	}

	if tcp, err := dialer.Dial("tcp", addr.String()); err != nil {
		return err
	} else {
		t.tcp = tcp
		t.ctx, t.cancel = context.WithCancel(context.Background())
		go recv(t.ctx, tcp, t.handler)
		return nil
	}
}

func (t *TCPClient) Write(data []byte) error {
	_, err := t.tcp.Write(data)
	return err
}
