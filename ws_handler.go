package network

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/gzjjyz/netlib/log"
	"net/http"
	"sync"
)

type wsOptions struct {
	MaxConnNum   int
	WriteChanCap int
	MaxMsgLen    uint32
	NewAgent     func(*WSConn) Agent
}

func (opt *wsOptions) Validation() error {
	if opt.MaxConnNum <= 0 {
		return errors.New("invalid MaxConnNum")
	}
	if opt.WriteChanCap <= 0 {
		return errors.New("invalid WriteChanCap")
	}
	if opt.MaxMsgLen <= 0 {
		return errors.New("invalid MaxMsgLen")
	}
	if opt.NewAgent == nil {
		return errors.New("newAgentHandler must not be nil")
	}
	if opt.MaxMsgLen <= 0 {
		return errors.New("invalid MaxMsgLen")
	}
	return nil
}

type WSHandler struct {
	opts       *wsOptions
	upgrader   websocket.Upgrader
	conns      WebsocketConnSet
	mutexConns sync.Mutex
	wg         sync.WaitGroup
}

func (handler *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}
	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug("upgrade error: %v", err)
		return
	}

	opts := handler.opts

	conn.SetReadLimit(int64(opts.MaxMsgLen))

	handler.wg.Add(1)
	defer handler.wg.Done()

	handler.mutexConns.Lock()
	if len(handler.conns) >= opts.MaxConnNum {
		handler.mutexConns.Unlock()
		conn.Close()
		log.Error("too many connections")
		return
	}
	handler.conns[conn] = struct{}{}
	handler.mutexConns.Unlock()

	wsConn := newWSConn(conn, opts.WriteChanCap, opts.MaxMsgLen)
	agent := opts.NewAgent(wsConn)
	agent.Run()

	// cleanup
	wsConn.Close()
	handler.mutexConns.Lock()
	delete(handler.conns, conn)
	handler.mutexConns.Unlock()
	agent.OnClose()
}
