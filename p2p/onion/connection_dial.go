package onion

import (
	"errors"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
)

func (c *Connection) handleConnectInternal(cmd *command.Command) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.ConnectInternal == nil {
		return errors.New("connect internal not passed")
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	info, err := c.DHT.FindPeer(ctx, cmd.Data.ConnectInternal.PeerId)
	if err != nil {
		return fmt.Errorf("couldn't find peer: %w", err)
	}

	err = c.Host.Connect(ctx, info)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	c.Logger.Log(log.LogLevelDebug, "Connected to peer: %v", info)

	stream, err := c.Host.NewStream(ctx, info.ID, ProtocolId)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	c.Logger.Log(log.LogLevelDebug, "Piping traffic")
	defer c.Logger.Log(log.LogLevelDebug, "Finished")
	go io.Copy(c.Conn, stream)
	io.Copy(stream, c.Conn)

	return nil
}
