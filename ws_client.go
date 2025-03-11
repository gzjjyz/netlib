package netlib

import (
	"errors"
	"fmt"
	"github.com/gzjjyz/netlib/log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func NewWSClient(
	address string,
	writeChanCap int,
	interval time.Duration,
	maxMsgLen uint32,
	handshakeTimeout time.Duration,
	newAgentHandler func(*WSConn) Agent,
) (*WSClient, error) {
	if newAgentHandler == nil {
		return nil, fmt.Errorf("newAgentHandler must not be nil")
	}

	if writeChanCap <= 0 {
		return nil, fmt.Errorf("invalid writeChanCap %d", writeChanCap)
	}

	client := &WSClient{
		Addr:            address,
		ConnectInterval: interval,
		WriteChanCap:    writeChanCap,
		MaxMsgLen:       maxMsgLen,
	}

	if client.ConnectInterval > 0 {
		client.AutoReconnect = true
	}
	client.HandshakeTimeout = handshakeTimeout

	client.closeFlag.Store(false)
	client.NewAgent = newAgentHandler

	return client, nil
}

type WSClient struct {
	sync.Mutex
	Addr             string
	ConnectInterval  time.Duration
	WriteChanCap     int
	MaxMsgLen        uint32
	HandshakeTimeout time.Duration
	AutoReconnect    bool
	NewAgent         func(*WSConn) Agent
	dialer           websocket.Dialer
	conn             *websocket.Conn
	wg               sync.WaitGroup
	closeFlag        atomic.Bool
}

func (client *WSClient) Start() {
	client.dialer = websocket.Dialer{
		HandshakeTimeout: client.HandshakeTimeout,
	}

	client.wg.Add(1)
	go client.connect()
}

func (client *WSClient) dial() (*websocket.Conn, error) {
	for {
		conn, _, err := client.dialer.Dial(client.Addr, nil)
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

func (client *WSClient) connect() {
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

	conn.SetReadLimit(int64(client.MaxMsgLen))

	wsConn := newWSConn(conn, client.WriteChanCap, client.MaxMsgLen)
	agent := client.NewAgent(wsConn)
	if agent == nil {
		wsConn.Close()
		return
	}
	agent.Run()

	// cleanup
	wsConn.Close()
	agent.OnClose()

	if client.AutoReconnect {
		time.Sleep(client.ConnectInterval)
		goto reconnect
	}
}

func (client *WSClient) Stop() {
	if client.closeFlag.Load() {
		return
	}

	client.closeFlag.Store(true)
	client.conn.Close()

	client.wg.Wait()
}
