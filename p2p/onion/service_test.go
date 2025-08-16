package onion_test

import (
	"context"
	"slices"
	"testing"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/testsuite"
	"github.com/RogueTeam/onion/utils"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/assert"
)

func Test_Integration(t *testing.T) {
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

		type Test struct {
			Name   string
			Action func(t *testing.T, svc *onion.Service)
		}
		tests := []Test{
			{
				Name: "External",
				Action: func(t *testing.T, svc *onion.Service) {
					assertions := assert.New(t)

					ctx, cancel := utils.NewContext()
					defer cancel()

					c, err := svc.Circuit(ctx, targets)
					assertions.Nil(err, "failed to prepare circuit")
					defer c.Close()

					maddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
					if !assertions.Nil(err, "failed to prepare maddr") {
						return
					}

					l, err := manet.Listen(maddr)
					if !assertions.Nil(err, "failed to listen") {
						return
					}
					defer l.Close()

					var payload = []byte("HELLO")
					go func() {
						conn, err := l.Accept()
						if !assertions.Nil(err, "failed to accept connection") {
							return
						}
						_, err = conn.Write(payload)
						assertions.Nil(err, "failed to write payload")
					}()

					conn, err := c.External(ctx, l.Multiaddr())
					if !assertions.Nil(err, "failed to dial to external") {
						return
					}
					defer conn.Close()

					var received = make([]byte, len(payload))
					_, err = conn.Read(received)
					if !assertions.Nil(err, "failed to receive from listener") {
						return
					}

					assertions.Equal(payload, received, "payload")
				},
			},
			{
				Name: "Basic HiddenService",
				Action: func(t *testing.T, svc *onion.Service) {
					assertions := assert.New(t)

					ctx, cancel := utils.NewContext()
					defer cancel()

					// Prepare listener
					c1, err := svc.Circuit(ctx, targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer c1.Close()

					hiddenPriv, err := identity.NewKey()
					if !assertions.Nil(err, "failed to generate identity") {
						return
					}

					svcSession, err := c1.Bind(ctx, hiddenPriv)
					if !assertions.Nil(err, "failed to bind hidden service") {
						return
					}
					defer svcSession.Close()

					// Prepare client
					t.Log("Preparing client")
					c2, err := svc.Circuit(ctx, targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer c2.Close()

					address, err := onion.HiddenAddressFromPrivKey(hiddenPriv)
					if !assertions.Nil(err, "failed to get address from priv key") {
						return
					}

					clientSession, err := c2.Dial(ctx, address)
					if !assertions.Nil(err, "failed to open client session") {
						return
					}
					defer clientSession.Close()

					t.Log("Testing connection")
					var payload = []byte("HELLO")
					go func() {
						conn, err := svcSession.Accept()
						if !assertions.Nil(err, "failed to accept connection") {
							return
						}
						defer conn.Close()

						_, err = conn.Write(payload)
						assertions.Nil(err, "failed to write payload")
					}()

					var recv = make([]byte, len(payload))
					conn, err := clientSession.Open(ctx)
					if !assertions.Nil(err, "failed to open client session") {
						return
					}
					defer conn.Close()

					_, err = conn.Read(recv)
					if !assertions.Nil(err, "failed to read payload") {
						return
					}

					assertions.Equal(payload, recv, "expecting a different payload")

					t.Logf("Received: %s", recv)
				},
			},
			{
				Name: "Discover HiddenService",
				Action: func(t *testing.T, svc *onion.Service) {
					assertions := assert.New(t)

					ctx, cancel := utils.NewContext()
					defer cancel()

					// Prepare listener
					serverCircuit, err := svc.Circuit(ctx, targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer serverCircuit.Close()

					hiddenPriv, err := identity.NewKey()
					if !assertions.Nil(err, "failed to generate identity") {
						return
					}

					svcSession, err := serverCircuit.Bind(ctx, hiddenPriv)
					if !assertions.Nil(err, "failed to bind hidden service") {
						return
					}
					defer svcSession.Close()

					// Prepare client
					t.Log("Preparing client")
					clientCircuit, err := svc.Circuit(ctx, targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer clientCircuit.Close()

					address, err := onion.HiddenAddressFromPrivKey(hiddenPriv)
					if !assertions.Nil(err, "failed to get address") {
						return
					}

					// time.Sleep(5 * time.Second)
					peers, err := clientCircuit.HiddenDHT(ctx, onion.CidFromData(address))
					if !assertions.Nil(err, "failed to find peers") {
						return
					}
					assertions.GreaterOrEqual(len(peers), 1, "no peers found")
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

				currentAddrs := func() (others []peer.AddrInfo) {
					for _, peer := range peers {
						others = append(others, peer.Peerstore().PeerInfo(peer.ID()))
					}
					return others
				}()

				clientPeerDht, err := dht.New(
					context.TODO(),
					client,
					dht.Mode(dht.ModeClient),
					dht.BootstrapPeers(currentAddrs...),
					dht.Datastore(datastore.NewMapDatastore()),
				)
				assertions.Nil(err, "failed to prepare client DHT")
				defer clientPeerDht.Close()

				clientSvc, err := onion.New(
					onion.DefaultConfig().
						WithHost(client).
						WithDHT(clientPeerDht),
				)
				assertions.Nil(err, "failed to prepare peer service")

				test.Action(t, clientSvc)

			})
		}
	})
}
