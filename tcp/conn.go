package tcp

import (
	"net"
	"sync"

	"github.com/gzjjyz/logger"
)

type Conn struct {
	conn  net.Conn
	lock  sync.Mutex
	close bool
	w     chan []byte
}

func (c *Conn) init(conn net.Conn, pending int) {
	c.w = make(chan []byte, pending)

	go func() {
		for b := range c.w {
			if b == nil {
				break
			}
			_, err := conn.Write(b)
			if err != nil {
				break
			}
		}

		conn.Close()
		c.lock.Lock()
		c.close = true
		c.lock.Unlock()
	}()
}

func (c *Conn) Destroy() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.destroy()
}

func (c *Conn) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.close {
		return
	}

	c.write(nil)
	c.close = true
}
func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) {
	if b == nil {
		return
	}

	c.write(b)
}

func (c *Conn) write(b []byte) {
	if len(c.w) == cap(c.w) {
		logger.LogError("close conn. channel is full")
		return
	}
	c.w <- b
}

func (c *Conn) destroy() {
	c.conn.(*net.TCPConn).SetLinger(0)
	c.conn.Close()

	if !c.close {
		close(c.w)
		c.close = true
	}
}
