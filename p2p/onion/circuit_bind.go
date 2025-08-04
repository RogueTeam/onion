package onion

import (
	"encoding/hex"
	"fmt"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/hashicorp/yamux"
	"github.com/libp2p/go-libp2p/core/crypto"
)

// Binds a hidden service based on a private key
func (c *Circuit) Bind(priv crypto.PrivKey) (session *yamux.Session, err error) {
	rawPub, err := crypto.MarshalPublicKey(priv.GetPublic())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	rawPubHash := command.DefaultHashAlgorithm().Sum(rawPub)
	rawSign, err := priv.Sign(rawPubHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	var bind = command.Command{
		Action: command.ActionBind,
		Data: command.Data{
			Bind: &command.Bind{
				PublicKey: hex.EncodeToString(rawPub),
				Signature: hex.EncodeToString(rawSign),
			},
		},
	}
	err = bind.Send(c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send bind: %w", err)
	}

	session, err = yamux.Server(c.Active, yamux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade yamux: %w", err)
	}
	return session, nil
}
