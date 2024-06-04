package protocol

import (
	"encoding/binary"
)

// SerializeType defines serialization type of Payload.
type SerializeType byte

const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota
	// ProtoBuffer for Payload.
	ProtoBuffer
)

// CompressType defines decompression type.
type CompressType byte

const (
	// None does not compress.
	None CompressType = iota
	// Gzip uses gzip compression.
	Gzip
)

type Message struct {
	*Header
	Payload []byte
}

func NewMessage() *Message {
	header := Header([8]byte{})

	return &Message{
		Header: &header,
	}
}

type Header [8]byte

// CompressType returns compression type of messages.
func (h *Header) CompressType() CompressType {
	return CompressType(h[0])
}

// SetCompressType sets the compression type.
func (h *Header) SetCompressType(ct CompressType) {
	h[0] = byte(ct)
}

// SerializeType returns serialization type of payload.
func (h *Header) SerializeType() SerializeType {
	return SerializeType(h[1])
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[1] = byte(st)
}

// ProtoId returns protocol id.
func (h *Header) ProtoId() uint16 {
	return binary.BigEndian.Uint16(h[2:])
}

// SetProtoId sets the protocol id.
func (h *Header) SetProtoId(id uint16) {
	binary.BigEndian.PutUint16(h[2:], id)
}

func (h *Header) DataLen() uint32 {
	return binary.BigEndian.Uint32(h[4:])
}

func (h *Header) SetDataLen(l uint32) {
	binary.BigEndian.PutUint32(h[4:8], l)
}

func (m *Message) Encode() []byte {
	data := make([]byte, len(m.Payload)+8)
	copy(data, m.Header[:])
	copy(data[8:], m.Payload)
	return data
}
