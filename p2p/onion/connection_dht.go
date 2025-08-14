package onion

import (
	"context"
	"errors"
	"fmt"

	"github.com/RogueTeam/onion/p2p/onion/message"
)

func (c *Connection) HiddenDHT(ctx context.Context, msg *message.Message) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if msg.Data.HiddenDHT == nil {
		return errors.New("dial not passed")
	}

	peers, err := c.DHT.FindProviders(ctx, msg.Data.HiddenDHT.Cid)
	if err != nil {
		return fmt.Errorf("failed to find providers for cid: %w", err)
	}

	var response = message.Message{
		Data: message.Data{
			HiddenDHTResponse: &message.HiddenDHTResponse{
				Peers: peers,
			},
		},
	}
	err = response.Send(ctx, c.Conn, DefaultSettings)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	return nil
}
