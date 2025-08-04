package onion

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	"github.com/hashicorp/yamux"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	p2pYamux "github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multihash"
)

func HiddenAddressFromPrivKey(priv crypto.PrivKey) (address string, err error) {
	return HiddenAddressFromPubKey(priv.GetPublic())
}

func HiddenAddressFromPubKey(pub crypto.PubKey) (address string, err error) {
	rawPub, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return address, fmt.Errorf("failed to marshal public key: %w", err)
	}

	rawAddress := command.DefaultHashAlgorithm().Sum(rawPub)
	return hex.EncodeToString(rawAddress), nil
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

	log.Println(RelayModeCidString)
	log.Println(OutsideModeCidString)
}

func createCID[T string | []byte](data T) (cid.Cid, error) {
	mh, err := multihash.Sum([]byte(data), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.DagCBOR, mh), nil
}

// Empty settings
var DefaultSettings = &command.Settings{}

type Config struct {
	// Difficulty of the Proof Of Work
	// Higher number will prevent span bots but will make harder to peers to use the node
	PowDifficulty uint64
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
}

// You could try to setup your own service instance by setting this fields but the
// "New" function is a plus helper for configuring everything
type Service struct {
	// Settings exposed to connected peers in order to successfully handshake and authenticate commands
	Settings command.Settings
	// Id of the peer
	ID peer.ID
	// Noise upgrader. Used for preventing relays from sniffing your traffic.
	Noise *noise.Transport
	// Host already binding to an address
	Host host.Host
	// DHT service. Configured entirely by you
	DHT *dht.IpfsDHT
	// Hidden services the application is serving as proxy
	HiddenServices *utils.Map[string, *yamux.Session]
}

const ProtocolId protocol.ID = "/onionp2p"

// Register the service into a existing host.Host.
// Check the docs of Config
func New(cfg Config) (s *Service, err error) {
	ctx, cancel := utils.NewContext()
	defer cancel()

	if cfg.Bootstrap {
		err = dhtutils.WaitForBootstrap(ctx, cfg.Host, cfg.DHT)
		if err != nil {
			return nil, fmt.Errorf("failed to bootstrap: %w", err)
		}
	}

	// Notify to the network the service is available
	err = cfg.DHT.Provide(ctx, RelayModeP2PCid, len(cfg.DHT.RoutingTable().ListPeers()) > 0)
	if err != nil {
		return nil, fmt.Errorf("failed to provide relay cid: %w", err)
	}
	if cfg.OutsideMode {
		err = cfg.DHT.Provide(ctx, OutsideModeP2PCid, len(cfg.DHT.RoutingTable().ListPeers()) > 0)
		if err != nil {
			return nil, fmt.Errorf("failed to provide outside node cid: %w", err)
		}
	}

	s = &Service{
		Settings: command.Settings{
			OutsideMode:   cfg.OutsideMode,
			PoWDifficulty: cfg.PowDifficulty,
		},
		ID:             cfg.Host.ID(),
		Host:           cfg.Host,
		DHT:            cfg.DHT,
		HiddenServices: new(utils.Map[string, *yamux.Session]),
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
