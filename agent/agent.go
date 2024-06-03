package agent

import "github.com/gzjjyz/netlib/protocol"

// Agent the network agent
type Agent interface {
	OnOpen()
	OnReceive(message *protocol.Message) error // receive net message
	OnClose()                                  // close net connection
}
