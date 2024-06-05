package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/netlib/protocol"
	"net/http"
	"sync"
)

type handler struct {
	opt       *Options
	connSet   *connSet
	msgParser *protocol.Parser
	u         websocket.Upgrader
}

func createHandler(opt *Options) *handler {
	return &handler{
		opt:       opt,
		connSet:   newConnSet(opt.MaxConns),
		msgParser: protocol.NewParser(opt.MaxMsgLen),
		u: websocket.Upgrader{
			HandshakeTimeout: opt.HttpTimeout,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	conn, err := h.u.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	if !h.connSet.add(conn) {
		conn.Close()
		logger.LogError("[%s] accept error: too many connections", h.opt.ServerName)
		return
	}
	defer h.connSet.remove(conn)

	conn.SetReadLimit(int64(h.opt.MaxMsgLen))

	wsConn := &Conn{conn: conn}
	netAgent, err := h.opt.AgentCreator(wsConn)
	if err != nil {
		logger.LogError("[%s] accept error: create agent failed. %v", h.opt.ServerName, err)
		return
	}
	defer netAgent.OnClose()

	wsConn.init(conn, h.opt.MaxWriteChannelCap)

	netAgent.OnOpen()

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

		err = netAgent.OnReceive(msg)
		if err != nil {
			logger.LogError("receive message error %v", err)
			return
		}
	}
}

type connSet struct {
	maxConns int
	conns    map[*websocket.Conn]struct{}
	lock     sync.Mutex
	wg       sync.WaitGroup
}

func newConnSet(maxConns int) *connSet {
	return &connSet{
		conns:    make(map[*websocket.Conn]struct{}),
		maxConns: maxConns,
	}
}

func (cs *connSet) add(conn *websocket.Conn) bool {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if len(cs.conns) >= cs.maxConns {
		return false
	}

	cs.conns[conn] = struct{}{}
	cs.wg.Add(1)
	return true
}

func (cs *connSet) remove(conn *websocket.Conn) {
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
