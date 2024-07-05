package transport

import (
	"context"
	"github.com/yangjiechina/avformat/libbufio"
	"github.com/yangjiechina/avformat/utils"
	"net"
	"runtime"
	"syscall"
)

const (
	DefaultTCPRecvBufferSize = 4096
)

// Handler 传输事件处理器，负责连接传输和断开连接的回调
type Handler interface {
	// OnConnected 返回收流缓冲区
	OnConnected(conn net.Conn) []byte

	// OnPacket 返回收流缓冲区
	OnPacket(conn net.Conn, data []byte) []byte

	OnDisConnected(conn net.Conn, err error)
}

// 函数回调
type handler struct {
	onConnected    func(conn net.Conn) []byte
	onPacket       func(conn net.Conn, data []byte) []byte
	onDisConnected func(conn net.Conn, err error)
}

func (h *handler) OnConnected(conn net.Conn) []byte {
	if h.onConnected != nil {
		return h.onConnected(conn)
	}

	return nil
}

func (h *handler) OnPacket(conn net.Conn, data []byte) []byte {
	if h.onPacket != nil {
		return h.onPacket(conn, data)
	}

	return nil
}

func (h *handler) OnDisConnected(conn net.Conn, err error) {
	if h.onDisConnected != nil {
		h.onDisConnected(conn, err)
	}
}

type ITransport interface {
	Bind(addr net.Addr) error

	Close()

	SetHandler(handler Handler)

	SetHandler2(onConnected func(conn net.Conn) []byte,
		onPacket func(conn net.Conn, data []byte) []byte,
		onDisConnected func(conn net.Conn, err error))

	ListenIP() string

	ListenPort() int
}

type transport struct {
	handler Handler
	ctx     context.Context
	cancel  context.CancelFunc

	listenIP   string
	listenPort int
}

func (t *transport) setListenAddr(addr net.Addr) {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		t.listenIP = tcpAddr.IP.String()
		t.listenPort = tcpAddr.Port
	} else if udpAddr, ok := addr.(*net.UDPAddr); ok {
		t.listenIP = udpAddr.IP.String()
		t.listenPort = udpAddr.Port
	} else {
		panic(addr)
	}
}

func (t *transport) SetHandler(handler Handler) {
	t.handler = handler
}

func (t *transport) SetHandler2(onConnected func(conn net.Conn) []byte, onPacket func(conn net.Conn, data []byte) []byte, onDisConnected func(conn net.Conn, err error)) {
	utils.Assert(t.handler == nil)
	t.SetHandler(&handler{
		onConnected,
		onPacket,
		onDisConnected,
	})
}

func (t *transport) Close() {
	t.handler = nil
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
}

func (t *transport) ListenPort() int {
	return t.listenPort
}

func (t *transport) ListenIP() string {
	return t.listenIP
}

type ReuseServer struct {
	transport
	ConcurrentNumber int
	EnableReuse      bool
}

func (r *ReuseServer) GetSetOptFunc() func(network, address string, c syscall.RawConn) error {
	if r.ComputeConcurrentNumber() > 1 {
		return SetReuseOpt
	}

	return nil
}

func (r *ReuseServer) ComputeConcurrentNumber() int {
	if runtime.GOOS == "darwin" || !r.EnableReuse {
		r.ConcurrentNumber = 1
	}

	r.ConcurrentNumber = libbufio.MaxInt(r.ConcurrentNumber, 1)
	return r.ConcurrentNumber
}
