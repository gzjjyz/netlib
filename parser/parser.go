/**
 * @Author: ChenJunJi
 * @Date: 2025/03/04
 * @Desc:
**/

package parser

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Parser struct {
	opts *Option
}

func NewMsgParser(opt *Option) (*Parser, error) {
	if err := opt.Validation(); err != nil {
		return nil, err
	}

	return &Parser{
		opts: opt,
	}, nil
}

func (p *Parser) Read(reader io.Reader) ([]byte, error) {
	msgLen, err := p.opts.readMsgLen(reader)
	if err != nil {
		return nil, err
	}

	// data
	buffer := make([]byte, msgLen)
	if _, err = io.ReadFull(reader, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (p *Parser) PackMsg(args ...[]byte) ([]byte, error) {
	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > p.opts.MaxMsgLen {
		return nil, fmt.Errorf("message too long, exceeded specified limit %d", p.opts.MaxMsgLen)
	} else if msgLen < p.opts.MinMsgLen {
		return nil, fmt.Errorf("message too short, exceeded specified limit %d", p.opts.MinMsgLen)
	}

	msg := make([]byte, uint32(p.opts.LenMsgLen)+msgLen)

	// write len
	switch p.opts.LenMsgLen {
	case 1:
		msg[0] = byte(msgLen)
	case 2:
		if p.opts.LittleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen))
		}
	case 4:
		if p.opts.LittleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen)
		} else {
			binary.BigEndian.PutUint32(msg, msgLen)
		}
	}

	// write data
	l := p.opts.LenMsgLen
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	return msg, nil
}
