package netlib

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gzjjyz/netlib/log"
	"github.com/gzjjyz/netlib/parser"
)

func NewTCPServer(
	address string,
	maxConnNum int,
	writeChanCap int,
	newAgentHandler func(*TCPConn) Agent,
	opts *parser.Option,
) (*TCPServer, error) {
	if newAgentHandler == nil {
		return nil, errors.New("newAgentHandler must not be nil")
	}
	if writeChanCap <= 0 {
		return nil, fmt.Errorf("invalid writeChanCap %d", writeChanCap)
	}

	if maxConnNum <= 0 {
		return nil, errors.New("invalid maxConnNum")
	}

	if opts == nil {
		opts = parser.DefaultOption()
	}
	p, err := parser.NewMsgParser(opts)
	if err != nil {
		return nil, err
	}
	server := &TCPServer{
		Addr:         address,
		MaxConnNum:   maxConnNum,
		WriteChanCap: writeChanCap,
		NewAgent:     newAgentHandler,
		conns:        make(ConnSet),
		msgParser:    p,
	}
	return server, nil
}

type TCPServer struct {
	Addr         string
	MaxConnNum   int
	WriteChanCap int
	NewAgent     func(*TCPConn) Agent
	ln           net.Listener
	conns        ConnSet
	mutexConns   sync.Mutex
	wgLn         sync.WaitGroup
	wgConns      sync.WaitGroup

	// msg parser
	msgParser *parser.Parser
}

func (server *TCPServer) Start() error {
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	server.ln = ln

	go server.run()

	return nil
}

func (server *TCPServer) Stop() {
	server.ln.Close()
	server.wgLn.Wait()

	server.mutexConns.Lock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.mutexConns.Unlock()
	server.wgConns.Wait()
}

func (server *TCPServer) run() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	var tempDelay time.Duration
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Error("accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		server.mutexConns.Lock()
		if len(server.conns) >= server.MaxConnNum {
			server.mutexConns.Unlock()
			conn.Close()
			log.Debug("too many connections")
			continue
		}

		tcpConn := newTCPConn(conn, server.WriteChanCap, server.msgParser)
		agent := server.NewAgent(tcpConn)
		if nil == agent {
			server.mutexConns.Unlock()
			tcpConn.Close()
			continue
		}

		server.conns[conn] = struct{}{}
		server.mutexConns.Unlock()

		server.wgConns.Add(1)

		go func() {
			agent.Run()

			// cleanup
			tcpConn.Close()
			server.mutexConns.Lock()
			delete(server.conns, conn)
			server.mutexConns.Unlock()
			agent.OnClose()

			server.wgConns.Done()
		}()
	}
}
