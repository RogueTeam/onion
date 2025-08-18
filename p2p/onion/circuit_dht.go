package onion

import (
	"context"
	"fmt"
	"log"

	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Performs a remote look up in the last node of the circuit. This prevent exposing your query to the DHT network
func (c *Circuit) HiddenDHT(ctx context.Context, cid cid.Cid) (peers []peer.AddrInfo, err error) {
	var req = message.Message{
		Data: message.Data{
			HiddenDHT: &message.HiddenDHT{
				Cid: cid,
			},
		},
	}
	err = req.Send(ctx, c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send external: %w", err)
	}

	var res message.Message
	err = res.Recv(c.Active, DefaultSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to recv response: %w", err)
	}

	if res.Data.HiddenDHTResponse == nil {
		return nil, fmt.Errorf("no hidden dht response found: %w", err)
	}

	peers = res.Data.HiddenDHTResponse.Peers
	for _, peer := range peers {
		log.Println(peer)
		c.Onion.DHT.ProviderStore().AddProvider(context.TODO(), cid.Bytes(), peer)
	}
	return peers, nil
}
