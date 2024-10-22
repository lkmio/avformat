package transport

import (
	"context"
	"github.com/lkmio/avformat/utils"
	"net"
)

type TCPClient struct {
	transport
	conn     net.Conn
	listener *net.TCPListener
}

func (t *TCPClient) Bind(addr net.Addr) error {
	return nil
}

func (t *TCPClient) Connect(local, remote *net.TCPAddr) (net.Conn, error) {
	dialer := net.Dialer{
		LocalAddr: local,
	}

	tcp, err := dialer.Dial("tcp", remote.String())
	if err != nil {
		t.Close()
		return nil, err
	}

	t.conn = tcp
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.setListenAddr(tcp.LocalAddr())
	return t.conn, nil
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
