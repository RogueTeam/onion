package onion

import (
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/multiformats/go-multiaddr"
)

// Connect to a remote service outside the onion network.
// Notice the last peer of the circuit chain should support external connections.
// You can check this by doing the proper filtering once you called ListPeers
func (c *Circuit) External(maddr multiaddr.Multiaddr) (conn net.Conn, err error) {
	var external = command.Command{
		Action: command.ActionExternal,
		Data: command.Data{
			External: &command.External{
				Address: maddr,
			},
		},
	}
	err = external.Send(c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send external: %w", err)
	}
	return c.Active, nil
}
