package websocket

import (
	"net"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/gzjjyz/logger"
)

type Conn struct {
	conn  *websocket.Conn
	close atomic.Bool
	w     chan []byte
}

func (c *Conn) init(conn *websocket.Conn, pending int) {
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
			if err := conn.WriteMessage(websocket.BinaryMessage, b); nil != err {
				return
			}
		}
	}()
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

func (c *Conn) Read() ([]byte, error) {
	_, b, err := c.conn.ReadMessage()
	return b, err
}

func (c *Conn) Close() {
	c.destroy()
}

func (c *Conn) destroy() {
	if c.close.Load() {
		return
	}
	c.conn.NetConn().(*net.TCPConn).SetLinger(0)
	c.conn.Close()

	c.close.Store(true)
	close(c.w)
}

func (c *Conn) write(b []byte) {
	if len(c.w) == cap(c.w) {
		logger.LogError("close conn. channel is full")
		c.destroy()
		return
	}
	c.w <- b
}
