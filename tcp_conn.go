package netlib

import (
	"github.com/gzjjyz/netlib/parser"
	"net"
	"sync/atomic"

	"github.com/gzjjyz/netlib/log"
)

type ConnSet map[net.Conn]struct{}

type TCPConn struct {
	conn      net.Conn
	writeChan chan []byte
	closeFlag atomic.Bool
	parser    *parser.Parser
}

func newTCPConn(conn net.Conn, writeChanCap int, msgParser *parser.Parser) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.conn = conn
	tcpConn.writeChan = make(chan []byte, writeChanCap)
	tcpConn.parser = msgParser

	go func() {
		for b := range tcpConn.writeChan {
			if b == nil {
				break
			}

			_, err := conn.Write(b)
			if err != nil {
				break
			}
		}

		conn.Close()
		tcpConn.closeFlag.Store(true)
	}()

	return tcpConn
}

func (tcpConn *TCPConn) doDestroy() {
	tcpConn.conn.(*net.TCPConn).SetLinger(0)
	tcpConn.conn.Close()

	if !tcpConn.closeFlag.Load() {
		close(tcpConn.writeChan)
		tcpConn.closeFlag.Store(true)
	}
}

func (tcpConn *TCPConn) Destroy() {
	tcpConn.doDestroy()
}

func (tcpConn *TCPConn) Close() {
	if tcpConn.closeFlag.Load() {
		return
	}

	tcpConn.doWrite(nil)
	tcpConn.closeFlag.Store(true)
}

func (tcpConn *TCPConn) doWrite(b []byte) {
	if len(tcpConn.writeChan) == cap(tcpConn.writeChan) {
		log.Debug("close conn: channel full")
		tcpConn.doDestroy()
		return
	}

	tcpConn.writeChan <- b
}

// b must not be modified by the others goroutines
func (tcpConn *TCPConn) Write(b []byte) {
	if b == nil {
		return
	}

	if tcpConn.closeFlag.Load() {
		return
	}

	tcpConn.doWrite(b)
}

func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b)
}

func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

func (tcpConn *TCPConn) ReadMsg() ([]byte, error) {
	return tcpConn.parser.Read(tcpConn)
}

func (tcpConn *TCPConn) WriteMsg(args ...[]byte) error {
	buf, err := tcpConn.parser.PackMsg(args...)
	if err != nil {
		return err
	}
	tcpConn.Write(buf)
	return nil
}
