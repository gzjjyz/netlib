package protocol

import (
	"encoding/binary"
)

const (
	CompressTypeBit  = 0 // 压缩类型
	SerializeTypeBit = 1 // 序列化类型
	MessageCmdBit    = 2 // 消息命令字
	PayloadSizeBit   = 4 // 消息体长度
	MessageHeaderLen = 8 // 消息长度
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
	return CompressType(h[CompressTypeBit])
}

// SetCompressType sets the compression type.
func (h *Header) SetCompressType(ct CompressType) {
	h[CompressTypeBit] = byte(ct)
}

// SerializeType returns serialization type of payload.
func (h *Header) SerializeType() SerializeType {
	return SerializeType(h[SerializeTypeBit])
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[SerializeTypeBit] = byte(st)
}

// Cmd returns protocol id.
func (h *Header) Cmd() uint16 {
	return binary.BigEndian.Uint16(h[MessageCmdBit:])
}

// SetCmd sets the protocol id.
func (h *Header) SetCmd(id uint16) {
	binary.BigEndian.PutUint16(h[MessageCmdBit:], id)
}

func (h *Header) DataLen() uint32 {
	return binary.BigEndian.Uint32(h[PayloadSizeBit:])
}

func (h *Header) SetDataLen(l uint32) {
	binary.BigEndian.PutUint32(h[PayloadSizeBit:], l)
}

func (m *Message) Encode() []byte {
	data := make([]byte, len(m.Payload)+MessageHeaderLen)
	copy(data, m.Header[:])
	copy(data[MessageHeaderLen:], m.Payload)
	return data
}
