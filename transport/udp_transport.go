package transport

import (
	"context"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"runtime"
	"syscall"
)

type UDPTransport struct {
	transportImpl
	udp             []net.PacketConn
	concurrentCount int
}

func reusePortControl(network, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		if runtime.GOOS != "darwin" {
			syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, 0x4, 1)
		}
	})
}

func NewUDPServer(addr net.Addr, handler Handler) (*UDPTransport, error) {
	count := runtime.NumCPU()
	if runtime.GOOS == "darwin" {
		count = 1
	}

	transport := &UDPTransport{concurrentCount: count}
	transport.SetHandler(handler)
	return transport, transport.Bind(addr)
}

func (u *UDPTransport) Bind(addr net.Addr) error {
	utils.Assert(u.handler != nil)

	u.ctx, u.cancel = context.WithCancel(context.Background())
	u.concurrentCount = libbufio.MaxInt(u.concurrentCount, 1)
	for i := 0; i < u.concurrentCount; i++ {
		lc := net.ListenConfig{
			Control: reusePortControl,
		}
		socket, err := lc.ListenPacket(u.ctx, "udp", addr.String())
		if err != nil {
			return err
		}

		u.udp = append(u.udp, socket)
		go recv2(u.ctx, socket, u.handler)
	}

	return nil
}

func (u *UDPTransport) Send(data []byte, addr net.Addr) (int, error) {
	return u.udp[0].WriteTo(data, addr)
}

func recv2(ctx context.Context, conn net.PacketConn, handler Handler) {
	bytes := make([]byte, 1500)

	for ctx.Err() == nil {
		n, addr, err := conn.ReadFrom(bytes)
		if err != nil {
			continue
		}

		if n > 0 && handler != nil {
			udpConn := &UDPConn{conn, conn.LocalAddr(), addr}
			handler.OnPacket(udpConn, bytes[:n])
		}
	}

	_ = conn.Close()
}
