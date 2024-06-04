package protocol

import (
	"errors"
	"io"
)

var (
	ErrMessageTooLong = errors.New("message is too long")
)

type Parser struct {
	maxMessageSize uint32       // max message size
	compressSize   int          // start compress size
	compressType   CompressType // compress type
}

func NewParser(maxMsgLen uint32) *Parser {
	return &Parser{
		maxMessageSize: maxMsgLen,
	}
}

func (p *Parser) Decode(r io.Reader) (*Message, error) {
	m := NewMessage()

	_, err := io.ReadFull(r, m.Header[:])
	if err != nil {
		return nil, err
	}

	l := m.DataLen()
	if p.maxMessageSize > 0 && l > p.maxMessageSize {
		return nil, ErrMessageTooLong
	}

	m.Payload = make([]byte, l)

	_, err = io.ReadFull(r, m.Payload)
	if err != nil {
		return nil, err
	}

	if m.CompressType() != None {
		// todo compress
	}

	return m, nil
}
