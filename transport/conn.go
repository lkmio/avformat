package transport

import (
	"net"
	"time"
)

// Conn 为连接句柄扩展public data字段
type Conn struct {
	conn net.Conn
	//接收缓冲区
	buffer []byte
	Data   interface{}

	closeCb func(conn net.Conn, err error)
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *Conn) Close() error {
	if c.closeCb != nil {
		c.closeCb(c, nil)
	}
	return c.conn.Close()
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
