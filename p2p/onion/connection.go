package onion

import (
	"context"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/RogueTeam/onion/utils"
	"github.com/hashicorp/yamux"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

// Unit responsible to handle a single stream to a peer
type Connection struct {
	// Our Host
	Host host.Host
	// Our DHT
	DHT *dht.IpfsDHT
	// net.Conn compatible interface of the stream or the noise channel
	Conn net.Conn
	// Our Settings
	Settings *message.Settings
	// Raw Stream of the connection
	Stream network.Stream
	// Logger for pretty printing
	Logger log.Logger
	// Noise channel using our real identity.
	// This allows the peers use untrusted nodes and validate
	// them to be who they are
	Noise *noise.Transport
	// No other msg can be used if the connection is not firstly
	// secured using the noise channel
	Secured bool
	// Used for identifying those peers that support External mode (Exit nodes)
	ExitNode bool
	// Storage for hidden services
	HiddenServices *utils.Map[peer.ID, *yamux.Session]
}

// Base logic for handling the connection
func (c *Connection) Handle(ctx context.Context) (err error) {
	// Send Settings
	var settings = message.Message{
		Data: message.Data{
			Settings: c.Settings,
		},
	}
	err = settings.Send(ctx, c.Conn, DefaultSettings)
	if err != nil {
		c.Logger.Log(log.LogLevelError, "SENDING SETTINGS: %v", err)
		return
	}
	//

	var msg message.Message
	for {
		err = msg.Recv(c.Conn, c.Settings)
		if err != nil {
			c.Logger.Log(log.LogLevelError, "READING MSG: %v", err)
			return
		}

		switch {
		case msg.Data.Noise != nil:
			err = c.UpgradeToNoise(&msg)
			if err != nil {
				return fmt.Errorf("failed to handle noise msg: %w", err)
			}
		case msg.Data.Extend != nil:
			err = c.Extend(&msg)
			if err != nil {
				return fmt.Errorf("failed to handle dial: %w", err)
			}
		case msg.Data.External != nil:
			err = c.External(&msg)
			if err != nil {
				return fmt.Errorf("failed to handle external: %w", err)
			}
		case msg.Data.Bind != nil:
			err = c.Bind(&msg)
			if err != nil {
				return fmt.Errorf("failed to handle bind: %w", err)
			}
		case msg.Data.Dial != nil:
			err = c.Dial(&msg)
			if err != nil {
				return fmt.Errorf("failed to handle bind: %w", err)
			}
		case msg.Data.HiddenDHT != nil:
			err = c.HiddenDHT(ctx, &msg)
			if err != nil {
				return fmt.Errorf("failed to handle hidden dht: %w", err)
			}
		default:
			return fmt.Errorf("invalid msg received")
		}
	}
}
