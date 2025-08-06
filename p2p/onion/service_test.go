package onion_test

import (
	"context"
	"slices"
	"testing"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/utils"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/assert"
)

func Test_Integration(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		const (
			ServicePeers = 10
		)

		assertions := assert.New(t)

		var dhts []*dht.IpfsDHT
		defer func() {
			t.Logf("Removing dhts: %d", len(dhts))
			for _, d := range dhts {
				d.Close()
			}
		}()
		var relays []*relay.Relay
		defer func() {
			t.Logf("Removing relays: %d", len(relays))
			for _, r := range relays {
				r.Close()
			}
		}()
		var peers []host.Host
		defer func() {
			t.Logf("Removing peers: %d", len(peers))
			for _, peer := range peers {
				peer.Close()
			}
		}()

		var svcs []*onion.Service
		for index := range ServicePeers {
			ident, err := identity.NewKey()
			if !assertions.Nil(err, "failed to prepare peer-1 key") {
				return
			}

			host, err := libp2p.New(
				libp2p.EnableRelay(),
				libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"),
				libp2p.Identity(ident),
			)
			if !assertions.Nil(err, "failed to prepare peer") {
				return
			}
			peers = append(peers, host)

			currentAddrs := func() (others []peer.AddrInfo) {
				for _, peer := range peers {
					if peer.ID() == host.ID() {
						continue
					}
					others = append(others, peer.Peerstore().PeerInfo(peer.ID()))
				}
				return others
			}()

			peerDht, err := dht.New(
				context.TODO(),
				host,
				dht.Mode(dht.ModeServer),
				dht.BootstrapPeers(currentAddrs...),
				dht.Datastore(datastore.NewMapDatastore()),
			)
			assertions.Nil(err, "failed to prepare DHT")
			dhts = append(dhts, peerDht)

			r, err := relay.New(host, relay.WithInfiniteLimits())
			assertions.Nil(err, "failed to prepare relay")
			relays = append(relays, r)

			svc, err := onion.New(onion.Config{
				Host:      host,
				DHT:       peerDht,
				Bootstrap: index != 0,
				ExitNode:  true,
			})
			assertions.Nil(err, "failed to prepare peer service")
			svcs = append(svcs, svc)
		}

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

					c, err := svc.Circuit(targets)
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

					conn, err := c.External(l.Multiaddr())
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

					// Prepare listener
					c1, err := svc.Circuit(targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer c1.Close()

					hiddenPriv, err := identity.NewKey()
					if !assertions.Nil(err, "failed to generate identity") {
						return
					}

					svcSession, err := c1.Bind(hiddenPriv)
					if !assertions.Nil(err, "failed to bind hidden service") {
						return
					}
					defer svcSession.Close()

					// Prepare client
					t.Log("Preparing client")
					c2, err := svc.Circuit(targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer c2.Close()

					address, err := onion.HiddenAddressFromPrivKey(hiddenPriv)
					if !assertions.Nil(err, "failed to get address from priv key") {
						return
					}

					clientSession, err := c2.Dial(address)
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
					conn, err := clientSession.Open()
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

					// Prepare listener
					serverCircuit, err := svc.Circuit(targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer serverCircuit.Close()

					hiddenPriv, err := identity.NewKey()
					if !assertions.Nil(err, "failed to generate identity") {
						return
					}

					svcSession, err := serverCircuit.Bind(hiddenPriv)
					if !assertions.Nil(err, "failed to bind hidden service") {
						return
					}
					defer svcSession.Close()

					// Prepare client
					t.Log("Preparing client")
					clientCircuit, err := svc.Circuit(targets)
					if !assertions.Nil(err, "failed to prepare circuit") {
						return
					}
					defer clientCircuit.Close()

					address, err := onion.HiddenAddressFromPrivKey(hiddenPriv)
					if !assertions.Nil(err, "failed to get address") {
						return
					}

					// time.Sleep(5 * time.Second)
					peers, err := clientCircuit.HiddenDHT(onion.CidFromData(address))
					if !assertions.Nil(err, "failed to find peers") {
						return
					}
					assertions.GreaterOrEqual(len(peers), 1, "no peers found")
				},
			},
			{
				Name: "Firewalled",
				Action: func(t *testing.T, svc *onion.Service) {
					assertions := assert.New(t)

					// Prepare new node
					ident, err := identity.NewKey()
					assertions.Nil(err, "failed to prepare identity")

					server, err := libp2p.New(
						libp2p.Identity(ident),
						libp2p.NoListenAddrs,
						libp2p.EnableRelay(),
					)
					assertions.Nil(err, "failed to prepare peer")

					currentAddrs := func() (others []peer.AddrInfo) {
						for _, peer := range peers {
							others = append(others, peer.Peerstore().PeerInfo(peer.ID()))
						}
						return others
					}()

					serverDht, err := dht.New(
						context.TODO(),
						server,
						dht.Mode(dht.ModeClient),
						dht.BootstrapPeers(currentAddrs...),
						dht.Datastore(datastore.NewMapDatastore()),
					)
					defer serverDht.Close()

					ctx, cancel := utils.NewContext()
					defer cancel()
					err = dhtutils.WaitForBootstrap(ctx, server, serverDht)
					assertions.Nil(err, "failed to wait for dht")

					// Find any relay able node
					ctx, cancel = utils.NewContext()
					defer cancel()
					providers, err := serverDht.FindProviders(ctx, onion.RelayNodeP2PCid)
					if !assertions.Nil(err, "failed to find relay providers") {
						return
					}
					if !assertions.GreaterOrEqual(len(providers), 1, "no providers found") {
						return
					}

					// Reserve address
					// This will only return .Addrs if the relay used has any public address.
					var reservation *client.Reservation
					for _, provider := range providers {
						if func() (found bool) {
							ctx, cancel := utils.NewContext()
							defer cancel()
							t.Logf("Reserving against: %v", provider)
							var err error
							reservation, err = client.Reserve(ctx, server, provider)
							if err == nil {
								reservation.Addrs = append(reservation.Addrs, provider.Addrs...)
								return true
							}

							return false
						}() {
							break
						}
					}
					assertions.NotNil(reservation, "failed to reserve remote")

					circuitAddr, err := multiaddr.NewMultiaddr("/p2p/" + reservation.Voucher.Relay.String() + "/p2p-circuit/p2p/" + reservation.Voucher.Peer.String())
					assertions.Nil(err, "failed to process circuit address")
					var circuitAddrs []multiaddr.Multiaddr
					for _, remote := range reservation.Addrs {
						circuitAddrs = append(circuitAddrs, remote.Encapsulate(circuitAddr))
					}

					var serverInfo = peer.AddrInfo{
						ID:    server.ID(),
						Addrs: circuitAddrs,
					}

					ctx, cancel = utils.NewContext()
					defer cancel()
					err = svc.Host.Connect(ctx, serverInfo)
					assertions.Nil(err, "failed to connect to server")

					t.Log(server.Addrs())
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
					libp2p.EnableRelay(),
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

				r, err := relay.New(client, relay.WithInfiniteLimits())
				assertions.Nil(err, "failed to prepare client relay")
				defer r.Close()

				clientSvc, err := onion.New(
					onion.DefaultConfig().
						WithHost(client).
						WithRelay(r).
						WithDHT(clientPeerDht),
				)
				assertions.Nil(err, "failed to prepare peer service")

				test.Action(t, clientSvc)

			})
		}
	})
}
