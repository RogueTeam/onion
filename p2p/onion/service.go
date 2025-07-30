package onion

import (
	"fmt"

	"github.com/RogueTeam/onion/p2p/dhtutils"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

// Empty settings
var DefaultSettings = &command.Settings{}

type Config struct {
	PowDifficulty uint64
	Host          host.Host
	DHT           *dht.IpfsDHT
	Bootstrap     bool
}

type Service struct {
	settings      command.Settings
	id            peer.ID
	incomingNoise *noise.Transport
	host          host.Host
	dht           *dht.IpfsDHT
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

	s = &Service{
		settings: command.Settings{
			PoWDifficulty: cfg.PowDifficulty,
		},
		id:   cfg.Host.ID(),
		host: cfg.Host,
		dht:  cfg.DHT,
	}

	s.incomingNoise, err = noise.New(
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
