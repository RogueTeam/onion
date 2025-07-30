package onion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type Circuit struct {
	currentPeer  peer.ID
	orderedPeers []peer.ID
	settings     map[peer.ID]*Settings
	service      *Service
	rootStream   network.Stream
	active       net.Conn
}

func (c *Circuit) String() (s string) {
	raw, _ := json.Marshal(c.orderedPeers)
	return string(raw)
}

func (c *Circuit) Subconnect(id peer.ID) (err error) {
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
	if c.rootStream == nil {
		ctx := context.TODO()

		c.rootStream, err = c.service.host.NewStream(ctx, id, ProtocolId)
		if err != nil {
			return fmt.Errorf("failed to connecto to root peer: %w", err)
		}

		conn = &Stream{Stream: c.rootStream}
	} else {
		var found bool
		oldSettings, found := c.settings[c.currentPeer]
		if !found {
			return errors.New("no settings found for current peer")
		}
		conn = c.active

		var connInternal = Command{
			Action: ActionConnectInternal,
			Data: Data{
				ConnectInternal: &ConnectInternal{
					PeerId: id,
				},
			},
		}
		err = connInternal.Send(conn, oldSettings)
		if err != nil {
			return fmt.Errorf("failed to send connect internal: %w", err)
		}
	}

	// Retrieve settings
	var settingsCmd Command
	err = settingsCmd.Recv(conn, &Settings{PoWDifficulty: 0})
	if err != nil {
		return fmt.Errorf("failed to receive settings msg: %w", err)
	}

	if settingsCmd.Action != ActionSettings || settingsCmd.Data.Settings == nil {
		return errors.New("invalid settings command received")
	}

	settings := settingsCmd.Data.Settings
	c.settings[id] = settings

	// Upgrade tunnel
	var noiseCmd = Command{
		Action: ActionNoise,
		Data: Data{
			Noise: &Noise{
				PeerPublicKey: pubKeyBytes,
			},
		},
	}
	err = noiseCmd.Send(conn, settings)
	if err != nil {
		return fmt.Errorf("failed to send noise request: %w", err)
	}

	ns, err := noise.New(ProtocolId, hiddenIdentity, []upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamux.DefaultTransport}})
	if err != nil {
		return fmt.Errorf("failed to prepare noise transport: %w", err)
	}

	ctx := context.TODO()
	c.active, err = ns.SecureOutbound(ctx, conn, id)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	c.currentPeer = id
	return nil
}

func (c *Circuit) Close() (err error) {
	if c.rootStream != nil {
		return c.rootStream.Close()
	}
	return nil
}

func (s *Service) Circuit(peers []peer.ID) (c *Circuit, err error) {
	if len(peers) == 0 {
		return nil, errors.New("no peers provided")
	}

	c = &Circuit{
		settings: make(map[peer.ID]*Settings),
		service:  s,
	}
	for _, peerId := range peers {
		err = c.Subconnect(peerId)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to peer: %s: %w", peerId, err)
		}
	}
	return c, nil
}
