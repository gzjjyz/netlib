package tcp

import (
	"net"
	"sync/atomic"

	"github.com/gzjjyz/logger"
)

type Conn struct {
	conn  net.Conn
	close atomic.Bool
	w     chan []byte
}

func (c *Conn) init(conn net.Conn, pending int) {
	c.w = make(chan []byte, pending)

	go func() {
		defer func() {
			conn.Close()
			c.close.Store(true)
		}()
		for b := range c.w {
			if b == nil {
				return
			}
			_, err := conn.Write(b)
			if err != nil {
				return
			}
		}
	}()
}

func (c *Conn) Close() {
	c.destroy()
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) {
	if b == nil {
		return
	}

	if c.close.Load() {
		return
	}

	c.write(b)
}

func (c *Conn) write(b []byte) {
	if len(c.w) == cap(c.w) {
		logger.LogError("close conn. channel is full")
		c.destroy()
		return
	}
	c.w <- b
}

func (c *Conn) destroy() {
	if c.close.Load() {
		return
	}

	c.conn.(*net.TCPConn).SetLinger(0)
	c.conn.Close()

	c.close.Store(true)
	close(c.w)
}
