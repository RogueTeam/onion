package database_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/testsuite"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_Database(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		const (
			ServicePeers = 10
		)

		_, peers, _, close := testsuite.SetupNetwork(t)
		defer close()

		targets := make([]peer.ID, 0, len(peers))
		for _, peer := range slices.Backward(peers) {
			targets = append(targets, peer.ID())
		}

		networkPeers := func() (others []peer.AddrInfo) {
			for _, peer := range peers {
				others = append(others, peer.Peerstore().PeerInfo(peer.ID()))
			}
			return others
		}()

		type Test struct {
			Name   string
			Action func(t *testing.T, svc *onion.Onion)
		}
		tests := []Test{
			{
				Name: "Basic HiddenService",
				Action: func(t *testing.T, svc *onion.Onion) {
					assertions := assert.New(t)

					db := database.New(database.Config{
						Onion:           svc,
						RefreshInterval: time.Second,
					})
					defer db.Close()

					x := db.All()
					t.Logf("Peers: %v", x)
					assertions.GreaterOrEqual(len(x), 1, "no peers found")
				},
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				assertions := assert.New(t)

				ident, err := identity.NewKey()
				if !assertions.Nil(err, "failed to prepare peer-1 key") {
					return
				}
				client, err := libp2p.New(
					libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"),
					libp2p.Identity(ident),
				)
				if !assertions.Nil(err, "failed to prepare client peer") {
					return
				}
				defer client.Close()

				clientPeerDht, err := dht.New(
					context.TODO(),
					client,
					dht.Mode(dht.ModeClient),
					dht.BootstrapPeers(networkPeers...),
					dht.Datastore(datastore.NewMapDatastore()),
				)
				assertions.Nil(err, "failed to prepare client DHT")
				defer clientPeerDht.Close()

				clientSvc, err := onion.New(
					onion.DefaultConfig().
						WithBootstrap(true).
						WithHost(client).
						WithDHT(clientPeerDht),
				)
				assertions.Nil(err, "failed to prepare peer service")

				test.Action(t, clientSvc)

			})
		}
	})
}
