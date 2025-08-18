package proxy_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/proxy"
	"github.com/RogueTeam/onion/testsuite"
	"github.com/RogueTeam/onion/utils"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_Proxy(t *testing.T) {
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
			Action func(t *testing.T, svc *onion.Service)
		}
		tests := []Test{
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

					var requestPayload = []byte("HIDDEN")
					httpHandler := http.NewServeMux()
					httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						w.Write(requestPayload)
					})
					go http.Serve(svcSession, httpHandler)

					// Prepare client
					t.Log("Preparing client")

					clientIdentity, err := identity.NewKey()
					assertions.Nil(err, "failed to prepare identity")

					client, err := libp2p.New(libp2p.Identity(clientIdentity), libp2p.NoListenAddrs)
					assertions.Nil(err, "failed to prepare client")

					clientDht, err := dht.New(context.TODO(), client,
						dht.Mode(dht.ModeClient),
						dht.BootstrapPeers(networkPeers...),
						dht.Datastore(datastore.NewMapDatastore()),
					)
					assertions.Nil(err, "failed to prepare dht")

					clientSvc, err := onion.New(onion.DefaultConfig().
						WithDHT(clientDht).
						WithHost(client).
						WithBootstrap(true).
						WithHiddenMode(true).
						WithExitNode(false))
					assertions.Nil(err, "failed to prepare service")

					l, err := net.Listen("tcp", "127.0.0.1:0")
					assertions.Nil(err, "failed to listen")
					defer l.Close()

					proxyUrl, _ := url.Parse("http://" + l.Addr().String())

					p := proxy.New(proxy.Config{
						CircuitLength:        3,
						Onion:                clientSvc,
						PeersRefreshInterval: time.Minute,
					})
					go p.Serve(l)

					address, err := onion.HiddenAddressFromPrivKey(hiddenPriv)
					if !assertions.Nil(err, "failed to get address from priv key") {
						return
					}
					t.Logf("> Raw Address: %v", address)

					cid := onion.CidFromData(address)
					t.Logf("> Expected CID: %v", cid)

					httpClient := http.Client{
						Transport: &http.Transport{
							Proxy: http.ProxyURL(proxyUrl),
						},
					}

					res, err := httpClient.Get("http://" + address.String() + ".libonion")
					assertions.Nil(err, "failed to get hidden service")
					t.Logf("STATUS CODE: %v", res.StatusCode)
					defer res.Body.Close()

					contents, err := io.ReadAll(res.Body)
					assertions.Nil(err, "failed to read body")
					t.Logf("BODY: %v", string(contents))
					assertions.Equal(requestPayload, contents, "responde doesn't match")
				},
			},
			{
				Name: "External service",
				Action: func(t *testing.T, svc *onion.Service) {
					assertions := assert.New(t)

					// Prepare listener
					server, err := net.Listen("tcp", "127.0.0.1:0")
					assertions.Nil(err, "failed to listen service")
					defer server.Close()

					var requestPayload = []byte("EXTERNAL")
					httpHandler := http.NewServeMux()
					httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						w.Write(requestPayload)
					})
					go http.Serve(server, httpHandler)

					// Prepare client
					t.Log("Preparing client")

					clientIdentity, err := identity.NewKey()
					assertions.Nil(err, "failed to prepare identity")

					client, err := libp2p.New(libp2p.Identity(clientIdentity), libp2p.NoListenAddrs)
					assertions.Nil(err, "failed to prepare client")

					clientDht, err := dht.New(context.TODO(), client,
						dht.Mode(dht.ModeClient),
						dht.BootstrapPeers(networkPeers...),
						dht.Datastore(datastore.NewMapDatastore()),
					)
					assertions.Nil(err, "failed to prepare dht")

					clientSvc, err := onion.New(onion.DefaultConfig().
						WithDHT(clientDht).
						WithHost(client).
						WithBootstrap(true).
						WithHiddenMode(true).
						WithExitNode(false))
					assertions.Nil(err, "failed to prepare service")

					l, err := net.Listen("tcp", "127.0.0.1:0")
					assertions.Nil(err, "failed to listen")
					defer l.Close()

					proxyUrl, _ := url.Parse("http://" + l.Addr().String())

					p := proxy.New(proxy.Config{
						CircuitLength:        3,
						Onion:                clientSvc,
						PeersRefreshInterval: time.Minute,
					})
					go p.Serve(l)

					address := server.Addr().String()
					t.Logf("> Raw Address: %v", address)

					httpClient := http.Client{
						Transport: &http.Transport{
							Proxy: http.ProxyURL(proxyUrl),
						},
					}

					res, err := httpClient.Get("http://" + address)
					assertions.Nil(err, "failed to get hidden service")
					t.Logf("STATUS CODE: %v", res.StatusCode)
					defer res.Body.Close()

					contents, err := io.ReadAll(res.Body)
					assertions.Nil(err, "failed to read body")
					t.Logf("BODY: %v", string(contents))
					assertions.Equal(requestPayload, contents, "responde doesn't match")
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
