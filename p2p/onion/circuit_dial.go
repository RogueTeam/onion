package onion

import (
	"fmt"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/hashicorp/yamux"
)

// Receives the DefaultHashAlgorithm of the public key of the hidden service and returns a yamux.Session
// The yamux session can create multiple dials to the same address using the session.Open method.
// The circuit should be constructed in order to force the last node be the one advertising the service.
// If not, the connection will fail
func (c *Circuit) Dial(address string) (session *yamux.Session, err error) {
	var dial = command.Command{
		Action: command.ActionDial,
		Data: command.Data{
			Dial: &command.Dial{
				Address: address,
			},
		},
	}
	err = dial.Send(c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send dial: %w", err)
	}

	session, err = yamux.Server(c.Active, yamux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to negotiate connection: %w", err)
	}

	return session, nil
}
