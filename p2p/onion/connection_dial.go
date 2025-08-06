package onion

import (
	"errors"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/hashicorp/yamux"
)

// Dial to a hidden service hosted by the machine
func (c *Connection) Dial(cmd *command.Command) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.Dial == nil {
		return errors.New("dial not passed")
	}

	svcSession, found := c.HiddenServices.Load(cmd.Data.Dial.Address)
	if !found {
		return errors.New("service not hosted by this node")
	}

	clientSession, err := yamux.Client(c.Conn, yamux.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to upgrade client conn: %w", err)
	}
	defer clientSession.Close()

	for {
		clientConn, err := clientSession.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept client connection: %w", err)
		}

		serviceConn, err := svcSession.Open()
		if err != nil {
			return fmt.Errorf("failed to open new connection: %w", err)
		}

		go func() {
			defer serviceConn.Close()
			go io.Copy(clientConn, serviceConn)
			io.Copy(serviceConn, clientConn)
		}()
	}
}
