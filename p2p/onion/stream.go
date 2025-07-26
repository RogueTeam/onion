package onion

import (
	"net"

	"github.com/libp2p/go-libp2p/core/network"
)

type Stream struct {
	network.Stream
}

func (s *Stream) LocalAddr() net.Addr  { return nil }
func (s *Stream) RemoteAddr() net.Addr { return nil }

var _ net.Conn = (*Stream)(nil)
