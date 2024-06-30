package transport

import (
	"context"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"runtime"
	"syscall"
)

type UDPServer struct {
	transport
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

func NewUDPServer(addr net.Addr, handler Handler) (*UDPServer, error) {
	count := runtime.NumCPU()
	if runtime.GOOS == "darwin" {
		count = 1
	}

	server := &UDPServer{concurrentCount: count}
	server.SetHandler(handler)
	return server, server.Bind(addr)
}

func (u *UDPServer) Bind(addr net.Addr) error {
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
		go recvUdp(u.ctx, socket, u.handler)
	}

	return nil
}

func (u *UDPServer) Send(data []byte, addr net.Addr) (int, error) {
	return u.udp[0].WriteTo(data, addr)
}

func (u *UDPServer) Close() {
	if len(u.udp) > 0 {
		u.udp[0].Close()
	}
	u.transport.Close()
}

func recvUdp(ctx context.Context, conn net.PacketConn, handler Handler) {
	bytes := make([]byte, 1500)
	//音视频UDP收流都使用jitter buffer处理, 难免还是会拷贝一次, 所以UDP不使用外部的读取缓冲区.
	for ctx.Err() == nil {
		n, addr, err := conn.ReadFrom(bytes)
		if err != nil {
			println(err.Error())

			if n == 0 {
				break
			}
		}

		if n > 0 && handler != nil {
			c := &Conn{conn: &UDPConn{conn, conn.LocalAddr(), addr}, closeCb: handler.OnDisConnected}
			handler.OnPacket(c, bytes[:n])
		}
	}

	_ = conn.Close()
}
