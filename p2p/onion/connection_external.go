package onion

import (
	"errors"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	manet "github.com/multiformats/go-multiaddr/net"
)

// Handle the connection to an external service
func (c *Connection) External(cmd *command.Command) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.External == nil {
		return errors.New("external not passed")
	}
	if !c.OutsideMode {
		return errors.New("this peer doesn't support external mode")
	}

	remote, err := manet.Dial(cmd.Data.External.Address)
	if err != nil {
		return fmt.Errorf("failed to dial external: %w", err)
	}
	defer remote.Close()

	c.Logger.Log(log.LogLevelDebug, "Piping traffic")
	defer c.Logger.Log(log.LogLevelDebug, "Finished")

	go io.Copy(c.Conn, remote)
	_, err = io.Copy(remote, c.Conn)
	if err != nil {
		return fmt.Errorf("failed to read from connection: %w", err)
	}
	return nil
}
