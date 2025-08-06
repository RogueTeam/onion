package onion

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/onion/command"
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
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
)

var DefaultMuxerUpgrader = []upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamuxp2p.DefaultTransport}}

func HiddenAddressFromPrivKey(priv crypto.PrivKey) (address peer.ID, err error) {
	return peer.IDFromPrivateKey(priv)
}

func HiddenAddressFromPubKey(pub crypto.PubKey) (address peer.ID, err error) {
	return peer.IDFromPublicKey(pub)
}

const (
	BaseString           = "onionp2p"
	RelayModeCidString   = BaseString + "-relay"
	OutsideModeCidString = BaseString + "-outsidenode"
)

var (
	RelayModeP2PCid   cid.Cid
	OutsideModeP2PCid cid.Cid
)

// Initialize the CIDs used to find onion relays
// These are hardcoded and area persistent cross boots
func init() {
	var err error
	RelayModeP2PCid, err = createCID(RelayModeCidString)
	if err != nil {
		log.Fatal(err)
	}

	OutsideModeP2PCid, err = createCID(OutsideModeCidString)
	if err != nil {
		log.Fatal(err)
	}

}

func createCID[T ~string | ~[]byte](data T) (cid.Cid, error) {
	bytes := []byte(data)

	mh, err := multihash.Sum(bytes, multihash.SHA3_512, -1)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to hash sum: %w", err)
	}
	return cid.NewCidV1(uint64(multicodec.DagCbor), mh), nil
}

// Empty settings
var DefaultSettings = &command.Settings{}

type Config struct {
	// LIBP2P host already listening and running
	Host host.Host
	// DHT instance already running
	DHT *dht.IpfsDHT
	// Run the bootstrap operation
	// When set DHT will Bootstrap and wait until there are nodes connected
	Bootstrap bool
	// Do not advertise this node.
	// Make sure to run this with a Client only DHT
	HiddenMode bool
	// Allow connections outside the network.
	// This basically connects the node into a proxy to the clearnet
	// Just like Tor's Exit nodes.
	OutsideMode bool
	// Time To Live
	TTL time.Duration
}

func (c Config) defaults() (cfg Config) {
	if c.TTL == 0 {
		c.TTL = time.Minute
	}
	return c
}

func (c Config) WithHost(host host.Host) (cfg Config) {
	c.Host = host
	return c
}

func (c Config) WithDHT(d *dht.IpfsDHT) (cfg Config) {
	c.DHT = d
	return c
}

func DefaultConfig() (cfg Config) {
	return Config{
		Bootstrap:   true,
		HiddenMode:  false,
		OutsideMode: false,
		TTL:         time.Minute,
	}
}

// You could try to setup your own service instance by setting this fields but the
// "New" function is a plus helper for configuring everything
type Service struct {
	// Counter of the number of active connections
	// This is used for calculating the PoW difficulty
	Connections atomic.Int64
	// Id of the peer
	ID peer.ID
	// Noise upgrader. Used for preventing relays from sniffing your traffic.
	Noise *noise.Transport
	// Host already binding to an address
	Host host.Host
	// DHT service. Configured entirely by you
	DHT *dht.IpfsDHT
	// Work in outside mode allowing connections outside the network
	OutsideMode bool
	// Hidden services the application is serving as proxy
	HiddenServices *utils.Map[peer.ID, *yamux.Session]
}

const ProtocolId protocol.ID = "/onionp2p/0.0.1"

// Settings exposed to connected peers in order to successfully handshake and authenticate commands
// defered s.Connection.Add(-1) should be called to ensure non impossible pow difficulty
func (s *Service) Settings() (settings *command.Settings) {
	k := s.Connections.Add(1)
	diff := hashcash.SqrtDifficulty(hashcash.DefaultHashAlgorithm(), k)
	log.Println("DIFF", k, diff)
	return &command.Settings{
		OutsideMode:   s.OutsideMode,
		PoWDifficulty: diff,
	}
}

func PromoteService(outsideMode bool, d *dht.IpfsDHT) (doContinue bool) {
	ctx, cancel := utils.NewContext()
	defer cancel()
	err := d.Provide(ctx, RelayModeP2PCid, len(d.RoutingTable().ListPeers()) > 0)
	if err != nil {
		log.Printf("failed to provide relay cid: %v", err)
		return false
	}

	if outsideMode {
		ctx, cancel := utils.NewContext()
		defer cancel()

		err := d.Provide(ctx, OutsideModeP2PCid, len(d.RoutingTable().ListPeers()) > 0)
		if err != nil {
			log.Printf("failed to provide outside node cid: %v", err)
			return false
		}
	}
	return true
}

// Register the service into a existing host.Host.
// Check the docs of Config
func New(cfg Config) (s *Service, err error) {
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
				doContinue := PromoteService(cfg.OutsideMode, cfg.DHT)
				if !doContinue {
					return
				}
				<-ticker.C
			}
		}()
	}

	s = &Service{
		OutsideMode:    cfg.OutsideMode,
		ID:             cfg.Host.ID(),
		Host:           cfg.Host,
		DHT:            cfg.DHT,
		HiddenServices: new(utils.Map[peer.ID, *yamux.Session]),
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
