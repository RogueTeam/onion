package service_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/p2p/service"
	"github.com/RogueTeam/onion/testsuite"
	"github.com/RogueTeam/onion/utils"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_Service(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		const (
			ServicePeers = 10
		)

		_, peerHosts, _, close := testsuite.SetupNetwork(t)
		defer close()

		targets := make([]peer.ID, 0, len(peerHosts))
		for _, peer := range slices.Backward(peerHosts) {
			targets = append(targets, peer.ID())
		}

		networkPeers := func() (others []peer.AddrInfo) {
			for _, peer := range peerHosts {
				others = append(others, peer.Peerstore().PeerInfo(peer.ID()))
			}
			return others
		}()

		type Test struct {
			Name   string
			Action func(t *testing.T, o *onion.Onion)
		}
		tests := []Test{
			{
				Name: "Basic HiddenService",
				Action: func(t *testing.T, o *onion.Onion) {
					assertions := assert.New(t)

					// Preparing client
					clientIdentity, err := identity.NewKey()
					assertions.Nil(err, "failed to create client identity")

					client, err := libp2p.New(libp2p.Identity(clientIdentity), libp2p.NoListenAddrs)
					assertions.Nil(err, "failed to prepare client")

					clientDht, err := dht.New(context.TODO(), client,
						dht.Mode(dht.ModeClient),
						dht.BootstrapPeers(append(networkPeers, o.Host.Peerstore().PeerInfo(o.ID))...),
						dht.Datastore(datastore.NewMapDatastore()),
					)
					assertions.Nil(err, "failed to prepare dht")

					clientOnion, err := onion.New(onion.DefaultConfig().
						WithDHT(clientDht).
						WithHost(client).
						WithBootstrap(true).
						WithHiddenMode(true).
						WithExitNode(false))
					assertions.Nil(err, "failed to prepare service")

					clientDb := database.New(database.Config{
						Onion:           clientOnion,
						RefreshInterval: time.Minute,
					})
					defer clientDb.Close()

					// Preparing server
					serverDb := database.New(database.Config{
						Onion:           o,
						RefreshInterval: time.Minute,
					})
					defer serverDb.Close()

					const replicas = 2
					svc := service.New(service.Config{
						Replicas:      replicas,
						CircuitLength: 3,
						Onion:         o,
						Database:      serverDb,
					})

					servicePrivKey, err := identity.NewKey()
					assertions.Nil(err, "failed to prepare service identity")
					hiddenAddr, err := onion.HiddenAddressFromPrivKey(servicePrivKey)
					assertions.Nil(err, "failed to obtain hidden address")

					l, err := svc.Listen(servicePrivKey)
					assertions.Nil(err, "failed to listen for hidden service")
					defer l.Close()

					// Test network
					circuitPeers, err := clientDb.Circuit(database.Circuit{Length: 3})
					if !assertions.Nil(err, "failed to get client circuit peers") {
						return
					}

					ctx, cancel := utils.NewContext()
					defer cancel()
					c, err := clientOnion.Circuit(ctx, circuitPeers)
					if !assertions.Nil(err, "failed to create circuit") {
						return
					}
					defer c.Close()

					const retries = 60
					var candidates []peer.AddrInfo
					for range retries {
						candidates, err = c.HiddenDHT(ctx, hiddenAddr)
						if !assertions.Nil(err, "failed to get candidates for hidden service") {
							return
						}
						t.Logf("CANDIDATES: %v", len(candidates))
						if len(candidates) == replicas {
							break
						}
						t.Log("Sleeping 1s. Allowing workers to spawn")
						time.Sleep(time.Second)
					}

					if !assertions.Len(candidates, replicas, "expected amount of candidates") {
						return
					}

					// Testing connection
					// - Preparing listener
					var payload = []byte("HELLO")
					go func() {
						conn, err := l.Accept()
						if !assertions.Nil(err, "failed to accept connection") {
							return
						}
						if !assertions.NotNil(conn, "connection should be not nil") {
							return
						}
						defer conn.Close()

						_, err = conn.Write(payload)
						assertions.Nil(err, "failed to write payload")
					}()

					t.Log("Sleeping 5s")
					time.Sleep(5 * time.Second)
					// Preparing consumer
					clientC, err := clientOnion.Circuit(ctx, []peer.ID{candidates[0].ID})
					assertions.Nil(err, "failed to prepare circuit")

					connector, err := clientC.Dial(ctx, hiddenAddr)
					assertions.Nil(err, "failed to dial")
					defer connector.Close()

					conn, err := connector.Open(ctx)
					assertions.Nil(err, "failed to open connection")
					defer conn.Close()

					var recv = make([]byte, len(payload))
					_, err = conn.Read(recv)
					assertions.Nil(err, "failed to read")
					assertions.Equal(payload, recv, "expecting other payload")

					t.Logf("RECEIVED %s", recv)
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
						WithHost(client).
						WithDHT(clientPeerDht),
				)
				assertions.Nil(err, "failed to prepare peer service")

				test.Action(t, clientSvc)

			})
		}
	})
}
