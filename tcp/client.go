package tcp

import (
	"github.com/gzjjyz/logger"
	"net"
	"sync/atomic"
	"time"
)

type Client struct {
	opt   *Options
	close atomic.Bool
}

func (c *Client) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", c.opt.Addr)
		if err == nil || c.closeFlag.Load() {
			return conn
		}

		logger.LogError("connect to %v error: %v", client.Addr, err)
		time.Sleep(client.ConnectInterval)
		continue
	}
}
