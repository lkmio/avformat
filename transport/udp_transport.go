package transport

import (
	"context"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"runtime"
	"syscall"
)

type UDPTransport struct {
	transportImpl
	udp []net.PacketConn
}

func reusePortControl(network, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		if runtime.GOOS != "darwin" {
			syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, 0x4, 1)
		}
	})
}

func (u *UDPTransport) Bind(addr net.Addr) error {
	utils.Assert(u.handler != nil)

	count := runtime.NumCPU()
	if runtime.GOOS == "darwin" {
		count = 1
	}

	u.ctx, u.cancel = context.WithCancel(context.Background())
	for i := 0; i < count; i++ {
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
