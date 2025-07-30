package onion_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"testing"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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
			rawIdent, _ := ident.Raw()
			t.Logf("Identity: %v", hex.EncodeToString(rawIdent))

			port := index + 8888
			host, err := libp2p.New(
				libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", port)),
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

			for _, info := range currentAddrs {
				if info.ID == host.ID() {
					continue
				}
				err = host.Connect(context.TODO(), info)
				assertions.Nil(err, "failed to connect to remote from host")
			}

			peerDht, err := dht.New(
				context.TODO(),
				host,
				dht.Mode(dht.ModeServer),
				dht.BootstrapPeers(currentAddrs...),
				dht.Datastore(datastore.NewMapDatastore()),
			)
			assertions.Nil(err, "failed to prepare DHT")
			dhts = append(dhts, peerDht)

			svc, err := onion.New(onion.Config{
				PowDifficulty: 1,
				Host:          host,
				DHT:           peerDht,
			})
			assertions.Nil(err, "failed to prepare peer service")
			svcs = append(svcs, svc)
		}

		ident, err := identity.NewKey()
		if !assertions.Nil(err, "failed to prepare peer-1 key") {
			return
		}
		client, err := libp2p.New(
			libp2p.ListenAddrStrings("/ip4/127.0.0.1/udp/9999/quic-v1"),
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

		for _, info := range currentAddrs {
			if info.ID == client.ID() {
				continue
			}
			err = client.Connect(context.TODO(), info)
			assertions.Nil(err, "failed to connect to remote from host")
		}

		clientPeerDht, err := dht.New(
			context.TODO(),
			client,
			dht.Mode(dht.ModeClient),
			dht.BootstrapPeers(currentAddrs...),
			dht.Datastore(datastore.NewMapDatastore()),
		)
		assertions.Nil(err, "failed to prepare client DHT")
		defer clientPeerDht.Close()

		targets := make([]peer.ID, 0, len(peers))
		for _, peer := range slices.Backward(peers) {
			targets = append(targets, peer.ID())
		}

		clientSvc, err := onion.New(onion.Config{
			PowDifficulty: 1,
			Host:          client,
			DHT:           clientPeerDht,
		})
		assertions.Nil(err, "failed to prepare peer service")

		c, err := clientSvc.Circuit(targets)
		assertions.Nil(err, "failed to prepare circuit")

		t.Logf("Circuit: %v", c)
	})
}
