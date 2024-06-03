package tcp

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/netlib/agent"
	"github.com/gzjjyz/netlib/protocol"
)

var AgentCreatorIsNil = errors.New("AgentCreator must not be nil")

type Server struct {
	opt       *Options
	connSet   *connSet
	msgParser *protocol.Parser

	wait sync.WaitGroup
	ln   net.Listener
}

func NewServer(opt *Options) (*Server, error) {
	opt.init()

	if opt.AgentCreator == nil {
		return nil, AgentCreatorIsNil
	}
	s := &Server{
		opt:       opt,
		connSet:   newConnSet(opt.MaxConns),
		msgParser: protocol.NewParser(opt.MaxMsgLen),
	}

	return s, s.init()
}

func (s *Server) init() error {
	ln, err := net.Listen("tcp", s.opt.Addr)
	if nil != err {
		return err
	}

	s.ln = ln

	return nil
}

func (s *Server) Start() {
	go s.run()
}

func (s *Server) Close() {
	s.ln.Close()

	s.wait.Wait()
	s.connSet.closeAll()
}

func (s *Server) run() {
	s.wait.Add(1)
	defer s.wait.Done()

	var delay time.Duration
	for {
		conn, err := s.ln.Accept()
		if nil != err {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if delay > time.Second {
					delay = time.Second
				}

				logger.LogError("accept error: %v; retrying in %v", err, delay)
				time.Sleep(delay)
				continue
			}
			return
		}
		delay = 0

		if !s.connSet.add(conn) {
			conn.Close()
			logger.LogError("[%s] accept error: too many connections", s.opt.ServerName)
			continue
		}

		tcpConn := &Conn{conn: conn}
		netAgent, err := s.opt.AgentCreator(tcpConn)
		if err != nil {
			conn.Close()
			s.connSet.remove(conn)
			logger.LogError("[%s] accept error: create a failed. %v", s.opt.ServerName, err)
			continue
		}

		go s.serveConn(conn, tcpConn, netAgent)
	}

}

func (s *Server) serveConn(conn net.Conn, tcpConn *Conn, agent agent.Agent) {
	defer func() {
		agent.OnClose()
		tcpConn.Close()
		s.connSet.remove(conn)
	}()

	tcpConn.init(conn, s.opt.MaxWriteChannelCap)

	agent.OnOpen()

	for {
		msg, err := s.msgParser.Decode(conn)
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

type connSet struct {
	maxConns int
	conns    map[net.Conn]struct{}
	lock     sync.Mutex
	wg       sync.WaitGroup
}

func newConnSet(maxConns int) *connSet {
	return &connSet{
		conns:    make(map[net.Conn]struct{}),
		maxConns: maxConns,
	}
}

func (cs *connSet) add(conn net.Conn) bool {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if len(cs.conns) >= cs.maxConns {
		return false
	}

	cs.conns[conn] = struct{}{}
	cs.wg.Add(1)
	return true
}

func (cs *connSet) remove(conn net.Conn) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if _, ok := cs.conns[conn]; ok {
		delete(cs.conns, conn)
		cs.wg.Done()
	}
}

func (cs *connSet) closeAll() {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	for conn := range cs.conns {
		conn.Close()
	}
	cs.conns = nil
	cs.wg.Wait()
}
