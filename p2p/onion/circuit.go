package onion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

// This instance represents a set of peers chained in order to hide you.
// Notice the only node that will know your real peer id is the first one.
// With the rest of them, your machine will use a fake identity.
// This will help prevent them to correlating in case of complicity. But maintain the ability
// To Zero trust communications.
type Circuit struct {
	// The last peer of the circuit.
	Current peer.ID
	// Ordered list of the peers chained into the circuit
	OrderedPeers []peer.ID
	// Settings for each peer of the circuit
	Settings map[peer.ID]*command.Settings
	// Back reference to the Service
	Service *Service
	// Root streaming used only for the first node of the circuit.
	RootStream network.Stream
	// The currently active connection.
	Active net.Conn
}

// String representation of a circuit
// Prints the list of peer ids used in the circuit
func (c *Circuit) String() (s string) {
	if len(c.OrderedPeers) == 0 {
		return "<empty>"
	}
	raw, _ := json.Marshal(c.OrderedPeers)
	return string(raw)
}

// Extends the circuit with a new peer.
// This function assumes the passed id corresponds to a valid onion protocol peer.
// Use ListPeers for more details
func (c *Circuit) Extend(id peer.ID) (err error) {
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
		ctx, _ := utils.NewContext()
		c.RootStream, err = c.Service.Host.NewStream(ctx, id, ProtocolId)
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

		var connInternal = command.Command{
			Action: command.ActionDial,
			Data: command.Data{
				ConnectInternal: &command.ConnectInternal{
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
	var settingsCmd command.Command
	err = settingsCmd.Recv(conn, DefaultSettings)
	if err != nil {
		return fmt.Errorf("failed to receive settings msg: %w", err)
	}

	if settingsCmd.Action != command.ActionSettings || settingsCmd.Data.Settings == nil {
		return errors.New("invalid settings command received")
	}

	settings := settingsCmd.Data.Settings
	c.Settings[id] = settings

	// Upgrade tunnel
	var noiseCmd = command.Command{
		Action: command.ActionNoise,
		Data: command.Data{
			Noise: &command.Noise{
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

	ctx, _ := utils.NewContext()
	c.Active, err = ns.SecureOutbound(ctx, conn, id)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	c.Current = id
	return nil
}

func (c *Circuit) Close() (err error) {
	if c.RootStream != nil {
		return c.RootStream.Close()
	}
	return nil
}

func (s *Service) Circuit(peers []peer.ID) (c *Circuit, err error) {
	if len(peers) == 0 {
		return nil, errors.New("no peers provided")
	}

	c = &Circuit{
		Settings: make(map[peer.ID]*command.Settings),
		Service:  s,
	}
	for _, peerId := range peers {
		err = c.Extend(peerId)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to peer: %s: %w", peerId, err)
		}
	}
	return c, nil
}
