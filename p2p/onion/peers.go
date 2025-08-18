package onion

import (
	"context"
	"fmt"

	"github.com/RogueTeam/onion/set"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer struct {
	Info  peer.AddrInfo
	Modes set.Set[cid.Cid]
}

// Lists peers compatible to the onion network.
// This function is useful with some filtering from your part.
// It returns a raw list of the peers using the onion protocol.
// You could filter based on public threats, remove possible fake nodes.
// Specific countries, etc.
func (o *Onion) ListPeers(ctx context.Context) (peers []*Peer, err error) {
	basicNodes, err := o.DHT.FindProviders(ctx, BasicNodeP2PCid)
	if err != nil {
		return nil, fmt.Errorf("failed to find basic mode peers: %w", err)
	}

	ref := make(map[peer.ID]*Peer)

	for _, info := range basicNodes {
		ref[info.ID] = &Peer{
			Info:  info,
			Modes: set.New(BasicNodeP2PCid),
		}
	}

	outsideMode, err := o.DHT.FindProviders(ctx, ExitNodeP2PCid)
	if err != nil {
		return nil, fmt.Errorf("failed to find outside mode peers: %w", err)
	}
	for _, info := range outsideMode {
		entry, found := ref[info.ID]
		if !found {
			continue
		}
		entry.Modes.Add(ExitNodeP2PCid)
	}

	peers = make([]*Peer, 0, len(ref))
	for _, entry := range ref {
		peers = append(peers, entry)
	}
	return peers, nil
}
