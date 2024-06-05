package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/netlib/agent"
	"github.com/gzjjyz/netlib/protocol"
)

type Client struct {
	opt    *Options
	conn   *websocket.Conn
	dialer websocket.Dialer
	close  bool
	wait   sync.WaitGroup
	lock   sync.Mutex
}

func NewClient(opt *Options) (*Client, error) {
	opt.init()
	if opt.AgentCreator == nil {
		return nil, AgentCreatorIsNil
	}
	c := &Client{opt: opt}

	return c, c.init()
}

func (c *Client) Start() {
	c.wait.Add(1)
	go c.connect()
}

func (c *Client) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.close {
		return
	}
	c.close = true

	if c.conn != nil {
		c.conn.Close()
	}

	c.wait.Wait()
}

func (c *Client) init() error {
	if c.opt.ConnectInterval <= 0 {
		c.opt.ConnectInterval = time.Second
		logger.LogInfo("invalid ConnectInterval, reset to %v", c.opt.ConnectInterval)
	}
	return nil
}

func (c *Client) dial() *websocket.Conn {
	for {
		conn, _, err := c.dialer.Dial(c.opt.Addr, nil)
		if err == nil || c.close {
			return conn
		}
		logger.LogError("connect to %v error: %v", c.opt.Addr, err)

		time.Sleep(c.opt.ConnectInterval)
		continue
	}
}

func (c *Client) connect() {
	defer c.wait.Done()
reconnect:
	conn := c.dial()
	if conn == nil {
		return
	}

	c.lock.Lock()
	c.conn = conn
	c.lock.Unlock()

	conn.SetReadLimit(int64(c.opt.MaxMsgLen))

	wsConn := &Conn{conn: conn}
	netAgent, err := c.opt.AgentCreator(wsConn)
	if err != nil {
		conn.Close()
		logger.LogError("[%s] connect error: create agent failed. %v", c.opt.Addr, err)
		return
	}

	wsConn.init(conn, c.opt.MaxWriteChannelCap)

	c.serveConn(conn, netAgent)

	if c.opt.AutoReconnect {
		time.Sleep(c.opt.ConnectInterval)
		goto reconnect
	}
}

func (c *Client) serveConn(conn *websocket.Conn, agent agent.Agent) {
	defer func() {
		agent.OnClose()
		conn.Close()
	}()
	agent.OnOpen()

	for {
		var msg *protocol.Message
		t, data, err := conn.ReadMessage()
		if err != nil {
			logger.LogError("receive message error %v", err)
			return
		}
		if websocket.BinaryMessage != t {
			logger.LogError("receive message but not binary")
			return
		}
		if len(data) < protocol.MessageHeaderLen {
			logger.LogError("receive message error %v", err)
			return
		}

		copy(msg.Header[:], data[:protocol.MessageHeaderLen])
		if size := msg.DataLen(); size > 0 {
			msg.Payload = make([]byte, size)
			copy(msg.Payload, data[protocol.MessageHeaderLen:])
		}

		err = agent.OnReceive(msg)
		if err != nil {
			logger.LogError("receive message error %v", err)
			return
		}
	}
}
