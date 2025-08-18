package onion

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

// Extends the circuit with a new peer.
// This function assumes the passed id corresponds to a valid onion protocol peer.
// Use ListPeers for more details
func (c *Circuit) Extend(ctx context.Context, id peer.ID) (err error) {
	// Generate a hidden Identifier to validate communications with the peer
	hiddenIdentity, err := identity.NewKey()
	if err != nil {
		return fmt.Errorf("failed to create hidden identity: %w", err)
	}

	pubKeyBytes, err := crypto.MarshalPublicKey(hiddenIdentity.GetPublic())
	if err != nil {
		return fmt.Errorf("failed to get public key bytes: %w", err)
	}

	var conn net.Conn
	// If there are no initial peer connected. New peer is then the root peer
	if c.RootStream == nil {
		c.RootStream, err = c.Onion.Host.NewStream(ctx, id, ProtocolId)
		if err != nil {
			return fmt.Errorf("failed to connecto to root peer: %w", err)
		}

		conn = &NetConnStream{Stream: c.RootStream}
	} else {
		var found bool
		oldSettings, found := c.Settings[c.Current]
		if !found {
			return errors.New("no settings found for current peer")
		}
		conn = c.Active

		var connInternal = message.Message{
			Data: message.Data{
				Extend: &message.Extend{
					PeerId: id,
				},
			},
		}
		err = connInternal.Send(ctx, conn, oldSettings)
		if err != nil {
			return fmt.Errorf("failed to send connect internal: %w", err)
		}
	}

	// Retrieve settings
	var settingsMsg message.Message
	err = settingsMsg.Recv(conn, DefaultSettings)
	if err != nil {
		return fmt.Errorf("failed to receive settings msg: %w", err)
	}

	if settingsMsg.Data.Settings == nil {
		return errors.New("invalid settings msg received")
	}

	settings := settingsMsg.Data.Settings
	c.Settings[id] = settings

	// Upgrade tunnel
	var noiseMsg = message.Message{
		Data: message.Data{
			Noise: &message.Noise{
				PeerPublicKey: pubKeyBytes,
			},
		},
	}
	err = noiseMsg.Send(ctx, conn, settings)
	if err != nil {
		return fmt.Errorf("failed to send noise request: %w", err)
	}

	ns, err := noise.New(ProtocolId, hiddenIdentity, []upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamux.DefaultTransport}})
	if err != nil {
		return fmt.Errorf("failed to prepare noise transport: %w", err)
	}

	c.Active, err = ns.SecureOutbound(ctx, conn, id)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	c.Current = id
	return nil
}
