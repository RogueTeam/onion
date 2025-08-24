package testsuite

import (
	"context"
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

func SetupNetwork(t *testing.T, servicePeers int) (dhts []*dht.IpfsDHT, peers []host.Host, svcs []*onion.Onion, close func()) {
	assertions := assert.New(t)

	close = func() {
		t.Logf("Removing dhts: %d", len(dhts))
		for _, d := range dhts {
			d.Close()
		}

		t.Logf("Removing peers: %d", len(peers))
		for _, peer := range peers {
			peer.Close()
		}
	}

	for index := range servicePeers {
		ident, err := identity.NewKey()
		if !assertions.Nil(err, "failed to prepare peer-1 key") {
			return
		}

		host, err := libp2p.New(
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

		svc, err := onion.New(onion.Config{
			Host:      host,
			DHT:       peerDht,
			Bootstrap: index != 0,
			ExitNode:  true,
		})
		assertions.Nil(err, "failed to prepare peer service")
		svcs = append(svcs, svc)
	}
	return dhts, peers, svcs, close
}
