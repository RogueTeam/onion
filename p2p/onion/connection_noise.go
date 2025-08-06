package onion

import (
	"errors"
	"fmt"

	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Upgrades the connection to use a noise channel
// If succeed output net.Conn is secured by encryption tunnel
func (c *Connection) UpgradeToNoise(msg *message.Message) (err error) {
	if msg.Data.Noise == nil {
		return errors.New("noise not passed")
	}

	pubKey, err := crypto.UnmarshalPublicKey(msg.Data.Noise.PeerPublicKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	peerId, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to get peer id from public key: %w", err)
	}

	ctx, _ := utils.NewContext()
	c.Conn, err = c.Noise.SecureInbound(ctx, c.Conn, peerId)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	c.Secured = true
	return nil
}
