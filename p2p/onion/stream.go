package onion

import (
	"net"

	"github.com/libp2p/go-libp2p/core/network"
)

// Utility function to use Streams as net.Conn compatible
type NetConnStream struct {
	network.Stream
}

// TODO: Somehow populate this
func (s *NetConnStream) LocalAddr() net.Addr  { return nil }
func (s *NetConnStream) RemoteAddr() net.Addr { return nil }

var _ net.Conn = (*NetConnStream)(nil)
