package onion

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/RogueTeam/onion/p2p/peers"
	"github.com/RogueTeam/onion/pow/hashcash"
	"github.com/RogueTeam/onion/utils"
	"github.com/hashicorp/yamux"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	p2pYamux "github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	yamuxp2p "github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

const LibOnionDnsFinale = ".onix"

var DefaultMuxerUpgrader = []upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamuxp2p.DefaultTransport}}

func HiddenAddressFromPrivKey(priv crypto.PrivKey) (address cid.Cid, err error) {
	return HiddenAddressFromPubKey(priv.GetPublic())
}

func HiddenAddressFromPubKey(pub crypto.PubKey) (address cid.Cid, err error) {
	peerId, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to get id from public key: %w", err)
	}

	return peer.ToCid(peerId), nil
}

const (
	BaseString         = "onionp2p"
	BasicNodeCidString = BaseString + "-basic"
	ExitNodeCidString  = BaseString + "-exitnode"
)

var (
	BasicNodeP2PCid cid.Cid = peers.IdentityCidFromData(BasicNodeCidString)
	ExitNodeP2PCid  cid.Cid = peers.IdentityCidFromData(ExitNodeCidString)
)

// Empty settings
var DefaultSettings = &message.Settings{}

// You could try to setup your own service instance by setting this fields but the
// "New" function is a plus helper for configuring everything
type Onion struct {
	// Counter of the number of active connections
	// This is used for calculating the PoW difficulty
	Connections atomic.Int64
	// Id of the peer
	ID peer.ID
	// Noise upgrader. Used for preventing node from sniffing your traffic.
	Noise *noise.Transport
	// Host already binding to an address
	Host host.Host
	// DHT service. Configured entirely by you
	DHT *dht.IpfsDHT
	// Work in outside mode allowing connections outside the network
	ExitNode bool
	// Hidden services the application is serving as proxy
	HiddenServices *utils.Map[cid.Cid, *yamux.Session]
}

const ProtocolId protocol.ID = "/onionp2p/0.0.1"

// Settings exposed to connected peers in order to successfully handshake and authenticate msgs
// defered s.Connection.Add(-1) should be called to ensure non impossible pow difficulty
func (o *Onion) Settings() (settings *message.Settings) {
	k := o.Connections.Add(1)
	diff := hashcash.LogDifficulty(hashcash.DefaultHashAlgorithm(), k)
	return &message.Settings{
		ExitNode:      o.ExitNode,
		PoWDifficulty: diff,
	}
}

func PromoteService(cfg *Config) (doContinue bool) {
	ctx, cancel := utils.NewContext()
	defer cancel()
	err := cfg.DHT.Provide(ctx, BasicNodeP2PCid, len(cfg.DHT.RoutingTable().ListPeers()) > 0)
	if err != nil {
		log.Printf("failed to provide basic cid: %v", err)
		return false
	}

	if cfg.ExitNode {
		ctx, cancel := utils.NewContext()
		defer cancel()

		err := cfg.DHT.Provide(ctx, ExitNodeP2PCid, len(cfg.DHT.RoutingTable().ListPeers()) > 0)
		if err != nil {
			log.Printf("failed to provide exit node cid: %v", err)
			return false
		}
	}
	return true
}

// Register the service into a existing host.Host.
// Check the docs of Config
func New(cfg Config) (s *Onion, err error) {
	cfg = cfg.defaults()

	ctx, cancel := utils.NewContext()
	defer cancel()

	if cfg.Bootstrap {
		err = dhtutils.WaitForBootstrap(ctx, cfg.Host, cfg.DHT)
		if err != nil {
			return nil, fmt.Errorf("failed to bootstrap: %w", err)
		}
	}

	// Notify to the network the service is available
	if !cfg.HiddenMode {
		go func() {
			ticker := time.NewTicker(cfg.TTL)
			defer ticker.Stop()

			for {
				doContinue := PromoteService(&cfg)
				if !doContinue {
					return
				}
				<-ticker.C
			}
		}()
	}

	s = &Onion{
		ExitNode:       cfg.ExitNode,
		ID:             cfg.Host.ID(),
		Host:           cfg.Host,
		DHT:            cfg.DHT,
		HiddenServices: new(utils.Map[cid.Cid, *yamux.Session]),
	}

	s.Noise, err = noise.New(
		ProtocolId,
		cfg.Host.Peerstore().PrivKey(cfg.Host.ID()),
		[]upgrader.StreamMuxer{{ID: ProtocolId, Muxer: p2pYamux.DefaultTransport}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare noise transport: %w", err)
	}

	// Register stream handler
	cfg.Host.SetStreamHandler(ProtocolId, s.StreamHandler)
	return s, nil
}
