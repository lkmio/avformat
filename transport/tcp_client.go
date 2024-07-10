package transport

import (
	"context"
	"github.com/lkmio/avformat/utils"
	"net"
)

type TCPClient struct {
	transport
	conn net.Conn
}

func (t *TCPClient) Bind(addr net.Addr) error {
	panic("please use the connect func")
}

func (t *TCPClient) Connect(local, addr *net.TCPAddr) error {
	dialer := net.Dialer{
		LocalAddr: local,
	}

	tcp, err := dialer.Dial("tcp", addr.String())
	if err != nil {
		t.Close()
		return err
	}

	t.conn = tcp
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.setListenAddr(tcp.LocalAddr())
	return nil
}

func (t *TCPClient) Receive() {
	utils.Assert(t.handler != nil)
	utils.Assert(t.conn != nil)

	recvTcp(t.ctx, t.conn, t.handler)
}

func (t *TCPClient) Write(data []byte) error {
	_, err := t.conn.Write(data)
	return err
}

func (t *TCPClient) Close() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}

	t.transport.Close()
}
