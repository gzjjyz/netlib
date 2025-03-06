package netlib

import (
	"crypto/tls"
	"github.com/gzjjyz/netlib/log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type WSServer struct {
	opts        *WSOptions
	addr        string
	httpTimeout time.Duration
	ln          net.Listener
	handler     *WSHandler
}

func NewWSServer(
	address string,
	timeout time.Duration,
	opts *WSOptions,
) (*WSServer, error) {
	if err := opts.Validation(); err != nil {
		return nil, err
	}
	s := &WSServer{
		addr:        address,
		httpTimeout: timeout,
		opts:        opts,
	}
	return s, s.init()
}

func (server *WSServer) init() error {
	ln, err := net.Listen("tcp", server.addr)
	if err != nil {
		return err
	}
	server.ln = ln

	return nil
}

func (server *WSServer) StartTLS(certFile, keyFile string) (err error) {
	config := &tls.Config{}
	config.NextProtos = []string{"http/1.1"}

	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	server.ln = tls.NewListener(server.ln, config)
	return nil
}

func (server *WSServer) Start() error {
	if server.httpTimeout <= 0 {
		server.httpTimeout = 10 * time.Second
		log.Info("invalid HTTPTimeout, reset to %v", server.httpTimeout)
	}

	server.handler = &WSHandler{
		opts:  server.opts,
		conns: make(WebsocketConnSet),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: server.httpTimeout,
			CheckOrigin:      func(_ *http.Request) bool { return true },
		},
	}

	httpServer := &http.Server{
		Addr:           server.addr,
		Handler:        server.handler,
		ReadTimeout:    server.httpTimeout,
		WriteTimeout:   server.httpTimeout,
		MaxHeaderBytes: 1024,
	}

	go httpServer.Serve(server.ln)
	return nil
}

func (server *WSServer) Close() {
	server.ln.Close()

	server.handler.mutexConns.Lock()
	for conn := range server.handler.conns {
		conn.Close()
	}
	server.handler.conns = nil
	server.handler.mutexConns.Unlock()

	server.handler.wg.Wait()
}
