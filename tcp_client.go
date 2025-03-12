package netlib

import (
	"errors"
	"fmt"
	"github.com/gzjjyz/netlib/log"
	"github.com/gzjjyz/netlib/parser"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func NewTCPClient(
	host string,
	writeChanCap int,
	interval time.Duration,
	newAgentHandler func(*TCPConn) Agent,
	opts *parser.Option,
) (*TCPClient, error) {
	if newAgentHandler == nil {
		return nil, fmt.Errorf("newAgentHandler must not be nil")
	}

	if writeChanCap <= 0 {
		return nil, fmt.Errorf("invalid writeChanCap %d", writeChanCap)
	}
	if opts == nil {
		opts = parser.DefaultOption()
	}

	p, err := parser.NewMsgParser(opts)
	if err != nil {
		return nil, err
	}

	client := &TCPClient{
		Addr:            host,
		ConnectInterval: interval,
		NewAgent:        newAgentHandler,
		WriteChanCap:    writeChanCap,
		msgParser:       p,
	}
	if client.ConnectInterval > 0 {
		client.AutoReconnect = true
	}
	client.closeFlag.Store(false)

	return client, nil
}

type TCPClient struct {
	sync.Mutex
	Addr            string
	ConnectInterval time.Duration
	WriteChanCap    int
	AutoReconnect   bool
	NewAgent        func(*TCPConn) Agent
	wg              sync.WaitGroup
	closeFlag       atomic.Bool

	conn net.Conn
	// msg parser
	msgParser *parser.Parser
}

func (client *TCPClient) Start() {
	client.wg.Add(1)
	go client.connect()
}

func (client *TCPClient) dial() (net.Conn, error) {
	for {
		conn, err := net.Dial("tcp", client.Addr)

		if client.closeFlag.Load() {
			return nil, errors.New("client closed")
		}

		if err != nil {
			log.Error("connect to %v error: %v", client.Addr, err)
			if !client.AutoReconnect {
				return nil, err
			}
			time.Sleep(client.ConnectInterval)

			continue
		}

		return conn, nil
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()

reconnect:
	conn, err := client.dial()
	if err != nil || conn == nil {
		return
	}

	if client.closeFlag.Load() {
		conn.Close()
		return
	}

	client.conn = conn

	tcpConn := newTCPConn(conn, client.WriteChanCap, client.msgParser)
	agent := client.NewAgent(tcpConn)
	if agent == nil {
		tcpConn.Close()
		return
	}
	agent.Run()

	// cleanup
	tcpConn.Close()
	agent.OnClose()

	if client.AutoReconnect {
		time.Sleep(client.ConnectInterval)
		goto reconnect
	}
}

func (client *TCPClient) Stop() {
	if client.closeFlag.Load() {
		return
	}
	client.closeFlag.Store(true)
	client.conn.Close()

	client.wg.Wait()
}
