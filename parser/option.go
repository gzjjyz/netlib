package parser

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type Option struct {
	LenMsgLen    int
	MinMsgLen    uint32
	MaxMsgLen    uint32
	LittleEndian bool
}

func DefaultOption() *Option {
	return &Option{
		LenMsgLen: 2,
		MinMsgLen: 0,
		MaxMsgLen: math.MaxUint16,
	}
}

func (opt *Option) Validation() error {
	var max uint32
	switch opt.LenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	default:
		return fmt.Errorf("parser option invalid LenMsgLen %v", opt.LenMsgLen)
	}
	if opt.MinMsgLen > max {
		opt.MinMsgLen = max
	}
	if opt.MaxMsgLen > max {
		opt.MaxMsgLen = max
	}
	return nil
}

func (opt *Option) readMsgLen(reader io.Reader) (msgLen uint32, err error) {
	var b [4]byte
	bufMsgLen := b[:opt.LenMsgLen]

	// read len
	if _, err = io.ReadFull(reader, bufMsgLen); err != nil {
		return 0, err
	}

	// parse len
	switch opt.LenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0])
	case 2:
		if opt.LittleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case 4:
		if opt.LittleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	// check len
	if msgLen > opt.MaxMsgLen {
		return 0, errors.New("message too long")
	} else if msgLen < opt.MinMsgLen {
		return 0, errors.New("message too short")
	}

	return msgLen, nil
}
