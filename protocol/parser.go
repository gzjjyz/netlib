package protocol

import (
	"errors"
	"io"
)

var (
	ErrMessageTooLong = errors.New("message is too long")
)

type Parser struct {
	maxMsgLen uint32
}

func NewParser(maxMsgLen uint32) *Parser {
	return &Parser{
		maxMsgLen: maxMsgLen,
	}
}

func (p *Parser) Decode(r io.Reader) (*Message, error) {
	m := NewMessage()

	_, err := io.ReadFull(r, m.Header[:])
	if err != nil {
		return nil, err
	}

	l := m.DataLen()
	if p.maxMsgLen > 0 && l > p.maxMsgLen {
		return nil, ErrMessageTooLong
	}

	m.data = make([]byte, l)

	_, err = io.ReadFull(r, m.data)
	if err != nil {
		return nil, err
	}

	if m.CompressType() != None {
		// todo compress
	}

	return m, nil
}
