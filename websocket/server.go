package websocket

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

var AgentCreatorIsNil = errors.New("AgentCreator must not be nil")

type Server struct {
	opt     *Options
	handler *handler

	ln net.Listener
}

func NewServer(opt *Options) (*Server, error) {
	opt.init()

	if opt.AgentCreator == nil {
		return nil, AgentCreatorIsNil
	}

	s := &Server{
		opt: opt,
	}

	return s, s.init()
}

func (s *Server) init() error {
	ln, err := net.Listen("tcp", s.opt.Addr)
	if err != nil {
		return err
	}

	if s.opt.CertFile != "" && s.opt.KeyFile != "" {
		config := &tls.Config{}
		config.NextProtos = []string{"http/1.1"}

		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(s.opt.CertFile, s.opt.KeyFile)

		if err != nil {
			return err
		}
		ln = tls.NewListener(ln, config)
	}

	s.ln = ln

	s.handler = createHandler(s.opt)

	return nil
}

func (s *Server) Start() {
	httpServer := &http.Server{
		Addr:           s.opt.Addr,
		Handler:        s.handler,
		ReadTimeout:    s.opt.HttpTimeout,
		WriteTimeout:   s.opt.HttpTimeout,
		MaxHeaderBytes: 1024,
	}
	httpServer.Serve(s.ln)
}

func (s *Server) Close() {
	s.ln.Close()
	s.handler.connSet.closeAll()
}
