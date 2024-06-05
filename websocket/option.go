package websocket

import (
	"github.com/gzjjyz/netlib/agent"
	"time"
)

type Options struct {
	// server name
	ServerName string
	// host:port address
	Addr string
	// CertFile file of tls cert
	CertFile string
	// KeyFile file of tls key
	KeyFile     string
	HttpTimeout time.Duration
	// max connection count
	// default 1024
	MaxConns int
	// max message length
	// default 1024
	MaxMsgLen uint32
	// max write channel capacity
	// default 1024
	MaxWriteChannelCap int
	// AgentCreator creates new agent
	AgentCreator    func(*Conn) (agent.Agent, error)
	AutoReconnect   bool
	ConnectInterval time.Duration
}

func (opt *Options) init() {
	if opt.Addr == "" {
		opt.Addr = "localhost:12306"
	}
	if opt.ServerName == "" {
		opt.ServerName = opt.Addr
	}
	if opt.MaxConns == 0 {
		opt.MaxConns = 1024
	}
	if opt.MaxMsgLen == 0 {
		opt.MaxMsgLen = 1024
	}
	if opt.MaxWriteChannelCap == 0 {
		opt.MaxWriteChannelCap = 1024
	}
	if opt.HttpTimeout == 0 {
		opt.HttpTimeout = time.Second * 3
	}
}
