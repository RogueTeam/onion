package onion

import (
	"errors"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Noise struct {
	PeerPublicKey []byte `json:"peerId"`
}

// Upgrades the connection to use a noise channel
// If succeed output net.Conn is secured by encryption tunnel
func (s *Service) handleNoise(cmd *command.Command, conn net.Conn) (secured net.Conn, err error) {
	if cmd.Data.Noise == nil {
		return nil, errors.New("noise not passed")
	}

	pubKey, err := crypto.UnmarshalPublicKey(cmd.Data.Noise.PeerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	peerId, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get peer id from public key: %w", err)
	}

	ctx, _ := utils.NewContext()
	secured, err = s.incomingNoise.SecureInbound(ctx, conn, peerId)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade connection: %w", err)
	}

	return secured, nil
}
