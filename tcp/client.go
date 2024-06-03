package tcp

import (
	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/netlib/agent"
	"github.com/gzjjyz/netlib/protocol"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	opt       *Options
	conn      net.Conn
	msgParser *protocol.Parser
	close     atomic.Bool
	wait      sync.WaitGroup
	lock      sync.Mutex
}

func NewClient(opt *Options) (*Client, error) {
	opt.init()

	if opt.AgentCreator == nil {
		return nil, AgentCreatorIsNil
	}

	c := &Client{
		opt:       opt,
		msgParser: protocol.NewParser(opt.MaxMsgLen),
	}

	return c, c.init()
}

func (c *Client) Start() {
	c.wait.Add(1)
	go c.connect()
}

func (c *Client) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.close.Store(true)
	c.conn.Close()

	c.wait.Wait()
}

func (c *Client) init() error {
	if c.opt.ConnectInterval <= 0 {
		c.opt.ConnectInterval = time.Second
		logger.LogInfo("invalid ConnectInterval, reset to %v", c.opt.ConnectInterval)
	}
	c.close.Store(false)

	return nil
}

func (c *Client) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", c.opt.Addr)
		if err == nil || c.close.Load() {
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

	if c.close.Load() {
		conn.Close()
		return
	}

	c.lock.Lock()
	c.conn = conn
	c.lock.Unlock()

	tcpConn := &Conn{conn: conn}
	netAgent, err := c.opt.AgentCreator(tcpConn)
	if err != nil {
		conn.Close()
		logger.LogError("[%s] connect error: create agent failed. %v", c.opt.Addr, err)
		return
	}
	c.serveConn(conn, tcpConn, netAgent)

	if c.opt.AutoReconnect {
		time.Sleep(c.opt.ConnectInterval)
		goto reconnect
	}
}

func (c *Client) serveConn(conn net.Conn, tcpConn *Conn, agent agent.Agent) {
	tcpConn.init(conn, c.opt.MaxWriteChannelCap)

	agent.OnOpen()

	for {
		msg, err := c.msgParser.Decode(conn)
		if err != nil {
			logger.LogError("read message error %v", err)
			break
		}

		err = agent.OnReceive(msg)
		if err != nil {
			logger.LogError("receive message error %v", err)
			break
		}
	}
}
