package transport

import (
	"context"
	"github.com/lkmio/avformat/utils"
	"net"
)

type UDPClient struct {
	transport
	udp *net.UDPConn
}

func (u *UDPClient) Bind(addr net.Addr) error {
	return nil
}

func (u *UDPClient) Connect(local, remote *net.UDPAddr) error {
	var udp *net.UDPConn
	var err error
	if remote == nil {
		udp, err = net.ListenUDP("udp", local)
	} else {
		udp, err = net.DialUDP("udp", local, remote)
	}

	if err != nil {
		u.Close()
		return err
	}

	u.udp = udp
	u.setListenAddr(udp.LocalAddr())
	return nil
}

func (u *UDPClient) Receive() {
	utils.Assert(u.handler != nil)
	u.ctx, u.cancel = context.WithCancel(context.Background())
	recvUdp(u.ctx, u.udp, u.handler)
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
	if u.udp != nil {
		u.udp.Close()
		u.udp = nil
	}
	u.transport.Close()
}
