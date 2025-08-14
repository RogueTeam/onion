package onion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
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
	Settings map[peer.ID]*message.Settings
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

func (c *Circuit) Close() (err error) {
	if c.RootStream != nil {
		return c.RootStream.Close()
	}
	return nil
}

func (s *Service) Circuit(ctx context.Context, peers []peer.ID) (c *Circuit, err error) {
	if len(peers) == 0 {
		return nil, errors.New("no peers provided")
	}

	c = &Circuit{
		Settings: make(map[peer.ID]*message.Settings),
		Service:  s,
	}
	for _, peerId := range peers {
		err = c.Extend(ctx, peerId)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to peer: %s: %w", peerId, err)
		}
	}
	return c, nil
}
