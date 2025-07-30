package onion

import (
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
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
	Settings command.Settings
	// Raw Stream of the connection
	Stream network.Stream
	// Logger for pretty printing
	Logger log.Logger
	// Noise channel using our real identity.
	// This allows the peers use untrusted nodes and validate
	// them to be who they are
	Noise *noise.Transport
	// No other command can be used if the connection is not firstly
	// secured using the noise channel
	Secured bool
}

// Base logic for handling the connection
func (c *Connection) Handle() (err error) {
	// Send Settings
	var settings = command.Command{
		Action: command.ActionSettings,
		Data: command.Data{
			Settings: &c.Settings,
		},
	}
	err = settings.Send(c.Conn, DefaultSettings)
	if err != nil {
		c.Logger.Log(log.LogLevelError, "SENDING SETTINGS: %v", err)
		return
	}
	//

	var cmd command.Command
	for {
		err = cmd.Recv(c.Conn, &c.Settings)
		if err != nil {
			c.Logger.Log(log.LogLevelError, "READING COMMAND: %v", err)
			return
		}

		switch cmd.Action {
		case command.ActionNoise:
			err = c.UpgradeToNoise(&cmd)
			if err != nil {
				return fmt.Errorf("failed to handle noise command: %w", err)
			}
		case command.ActionDial:
			err = c.ConnectInternal(&cmd)
			if err != nil {
				return fmt.Errorf("failed to handle connect internal: %w", err)
			}
		case command.ActionExternal:
			// TODO: Connect to PROTOCOL IP:PORT
			break
		default:
			return fmt.Errorf("unknown command: %s", cmd.Action.String())
		}
	}
}
