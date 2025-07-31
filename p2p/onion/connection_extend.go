package onion

import (
	"errors"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
)

// Connect to other peer inside the onion network. Used for extending existing Circuits
func (c *Connection) Extend(cmd *command.Command) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.Extend == nil {
		return errors.New("extend not passed")
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	c.Logger.Log(log.LogLevelDebug, "Connected to peer: %v", cmd.Data.Extend.PeerId)

	// By its own this connection can be seen in plaintext.
	// Its important to always upgrade to Noise channel
	stream, err := c.Host.NewStream(ctx, cmd.Data.Extend.PeerId, ProtocolId)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	c.Logger.Log(log.LogLevelDebug, "Piping traffic")
	defer c.Logger.Log(log.LogLevelDebug, "Finished")
	go io.Copy(c.Conn, stream)
	_, err = io.Copy(stream, c.Conn)
	if err != nil {
		return fmt.Errorf("failed to copy from conn: %w", err)
	}

	return nil
}
