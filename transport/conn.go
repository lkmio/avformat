package transport

import (
	"context"
	"net"
	"time"
)

// Conn 为连接句柄扩展public data字段
type Conn struct {
	conn             net.Conn
	buffer           []byte      //接收缓冲区
	pendingSendQueue chan []byte //等待发送的数据队列
	cancelFunc       func()
	cancelCtx        context.Context

	Data    interface{}                    //绑定参数
	closeCb func(conn net.Conn, err error) //主动调用Close时回调
	active  bool
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *Conn) doAsyncWrite() {
	for {
		select {
		case <-c.cancelCtx.Done():
			return
		case data := <-c.pendingSendQueue:
			c.conn.Write(data)
			break
		}
	}
}

func (c *Conn) EnableAsyncWriteMode(queueSize int) {
	c.pendingSendQueue = make(chan []byte, queueSize)
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())
	go c.doAsyncWrite()
}

type ZeroWindowSizeError struct {
}

func (z ZeroWindowSizeError) Error() string {
	return "zero window size"
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if c.cancelCtx != nil {
		select {
		case c.pendingSendQueue <- b:
			return len(b), nil
		default:
			return 0, &ZeroWindowSizeError{}
		}
	} else {
		return c.conn.Write(b)
	}
}

func (c *Conn) IsActive() bool {
	return c.active
}

func (c *Conn) Close() error {
	c.active = false

	err := c.conn.Close()

	if c.closeCb != nil {
		c.closeCb(c, nil)
	}

	if c.cancelCtx != nil {
		c.cancelFunc()
	}
	return err
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *Conn) ReallocateRecvBuffer(size int) {
	c.buffer = make([]byte, size)
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn, active: true}
}
