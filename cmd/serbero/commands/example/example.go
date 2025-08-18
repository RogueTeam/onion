package example

import (
	"time"

	"github.com/RogueTeam/onion/cmd/serbero/commands/node"
	"github.com/multiformats/go-multiaddr"
)

var ExampleConfig = node.Config{
	// AdvertiseAddresses: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/0.0.0.0/udp/8888/quic-v1")},
	ListenAddresses:  []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/0.0.0.0/udp/0/quic-v1")},
	TTL:              time.Minute,
	IdentityLocation: "serbero.id",
	HiddenMode:       false,
	ExitNode:         false,
	Bootstrap: &node.Bootstrap{
		Wait: true,
	},
	Proxy: &node.Proxy{
		CircuitLength: 3,
		ListenAddress: multiaddr.StringCast("/ip4/127.0.0.1/tcp/0"),
	},
}
