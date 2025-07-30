package onion

import (
	"fmt"
	"log"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multihash"
)

const (
	BaseString           = "onionp2p"
	RelayModeCidString   = BaseString + "-relay"
	OutsideModeCidString = BaseString + "-outsidenode"
)

var (
	RelayModeP2PCid   cid.Cid
	OutsideModeP2PCid cid.Cid
)

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
	PowDifficulty uint64
	Host          host.Host
	DHT           *dht.IpfsDHT
	Bootstrap     bool
	OutsideMode   bool
}

type Service struct {
	Settings command.Settings
	ID       peer.ID
	Noise    *noise.Transport
	Host     host.Host
	DHT      *dht.IpfsDHT
}

const ProtocolId protocol.ID = "/onionp2p"

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
		ID:   cfg.Host.ID(),
		Host: cfg.Host,
		DHT:  cfg.DHT,
	}

	s.Noise, err = noise.New(
		ProtocolId,
		cfg.Host.Peerstore().PrivKey(cfg.Host.ID()),
		[]upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamux.DefaultTransport}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare noise transport: %w", err)
	}

	// Register stream handler
	cfg.Host.SetStreamHandler(ProtocolId, s.StreamHandler)
	return s, nil
}
