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
	OnConnected(conn net.Conn)

	OnPacket(conn net.Conn, data []byte)

	OnDisConnected(conn net.Conn, err error)
}

type handlerImpl struct {
	onConnected    func(conn net.Conn)
	onPacket       func(conn net.Conn, data []byte)
	onDisConnected func(conn net.Conn, err error)
}

func (h *handlerImpl) OnConnected(conn net.Conn) {
	if h.onConnected != nil {
		h.onConnected(conn)
	}
}

func (h *handlerImpl) OnPacket(conn net.Conn, data []byte) {
	if h.onPacket != nil {
		h.onPacket(conn, data)
	}
}

func (h *handlerImpl) OnDisConnected(conn net.Conn, err error) {
	if h.onDisConnected != nil {
		h.onDisConnected(conn, err)
	}
}

type ITransport interface {
	Bind(addr net.Addr) error

	Close()

	SetHandler(handler Handler)

	SetHandler2(onConnected func(conn net.Conn),
		onPacket func(conn net.Conn, data []byte),
		onDisConnected func(conn net.Conn, err error))
}

type transportImpl struct {
	handler Handler

	ctx    context.Context
	cancel context.CancelFunc
}

func (impl *transportImpl) SetHandler(handler Handler) {
	impl.handler = handler
}

func (impl *transportImpl) SetHandler2(onConnected func(conn net.Conn), onPacket func(conn net.Conn, data []byte), onDisConnected func(conn net.Conn, err error)) {
	utils.Assert(impl.handler == nil)
	impl.SetHandler(&handlerImpl{
		onConnected,
		onPacket,
		onDisConnected,
	})
}

func (impl *transportImpl) Close() {
	impl.cancel()
}
