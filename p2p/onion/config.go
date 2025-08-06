package onion

import (
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

type Config struct {
	// LIBP2P host already listening and running
	Host host.Host
	// DHT instance already running
	DHT *dht.IpfsDHT
	// Run the bootstrap operation
	// When set DHT will Bootstrap and wait until there are nodes connected
	Bootstrap bool
	// Do not advertise this node.
	// Make sure to run this with a Client only DHT
	HiddenMode bool
	// Allow connections outside the network.
	// This basically connects the node into a proxy to the clearnet
	// Just like Tor's Exit nodes.
	ExitNode bool
	// Relay mode allows nodes without a public IP to also participate in the routing process.
	// When a relay is passed the engine assumes the incoming connections are allowed
	Relay *relay.Relay
	// Time To Live
	TTL time.Duration
}

func (c Config) WithRelay(r *relay.Relay) (cfg Config) {
	c.Relay = r
	return c
}

func (c Config) defaults() (cfg Config) {
	if c.TTL == 0 {
		c.TTL = time.Minute
	}
	return c
}

func (c Config) WithTTL(d time.Duration) (cfg Config) {
	c.TTL = d
	return c
}

func (c Config) WithHost(host host.Host) (cfg Config) {
	c.Host = host
	return c
}

func (c Config) WithDHT(d *dht.IpfsDHT) (cfg Config) {
	c.DHT = d
	return c
}

func DefaultConfig() (cfg Config) {
	return Config{
		Bootstrap:  true,
		HiddenMode: false,
		ExitNode:   false,
		TTL:        time.Minute,
	}
}
