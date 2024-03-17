package transport

import (
	"context"
	"net"
)

type TCPClient struct {
	transportImpl
}

func (t *TCPClient) Bind(addr net.Addr) error {
	panic("please use the connect func")
}

func (t *TCPClient) Connect(local, addr *net.TCPAddr) error {
	if tcp, err := net.DialTCP("tcp", local, addr); err != nil {
		return err
	} else {
		t.ctx, t.cancel = context.WithCancel(context.Background())
		go recv(t.ctx, tcp, t.handler)
		return nil
	}
}
