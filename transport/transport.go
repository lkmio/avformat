package transport

import (
	"context"
	"github.com/yangjiechina/avformat/utils"
	"net"
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
}

type transport struct {
	handler Handler

	ctx    context.Context
	cancel context.CancelFunc
}

func (impl *transport) SetHandler(handler Handler) {
	impl.handler = handler
}

func (impl *transport) SetHandler2(onConnected func(conn net.Conn) []byte, onPacket func(conn net.Conn, data []byte) []byte, onDisConnected func(conn net.Conn, err error)) {
	utils.Assert(impl.handler == nil)
	impl.SetHandler(&handler{
		onConnected,
		onPacket,
		onDisConnected,
	})
}

func (impl *transport) Close() {
	impl.handler = nil
	impl.cancel()
}
