package transport

import (
	"context"
	"net"
)

type UDPClient struct {
	transportImpl
	udp *net.UDPConn
}

func (u *UDPClient) Bind(addr net.Addr) error {
	return nil
}

func (u *UDPClient) Connect(local, remote *net.UDPAddr) error {
	udp, err := net.DialUDP("udp", local, remote)
	if err != nil {
		return err
	}

	u.udp = udp
	return nil
}

func (u *UDPClient) Recv() {
	u.ctx, u.cancel = context.WithCancel(context.Background())
	go recv2(u.ctx, u.udp, u.handler)
}

func (u *UDPClient) Write(data []byte) error {
	_, err := u.udp.Write(data)
	return err
}

func (u *UDPClient) WriteTo(data []byte, addr *net.UDPAddr) error {
	_, err := u.udp.WriteToUDP(data, addr)
	return err
}

func (u *UDPClient) Close() {
	u.transportImpl.Close()
	u.udp.Close()
}
