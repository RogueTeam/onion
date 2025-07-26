package onion

import (
	"fmt"
	"log"
	"net"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type Config struct {
	PowDifficulty uint64
	Host          host.Host
	DHT           *dht.IpfsDHT
}

type Service struct {
	powDifficulty uint64
	id            peer.ID
	incomingNoise *noise.Transport
	host          host.Host
	dht           *dht.IpfsDHT
}

const ProtocolId protocol.ID = "/onionp2p"

// Handles the stream
// On any error the stream is closed
func (s *Service) StreamHandler(stream network.Stream) {
	defer stream.Close()

	// Send Settings
	var settings = Command{
		Action: ActionSettings,
		Data: Data{
			Settings: &Settings{
				PoWDifficulty: s.powDifficulty,
			},
		},
	}
	err := settings.Send(stream, 0)
	if err != nil {
		log.Println("ERROR: SENDING SETTINGS:", err)
		return
	}
	//

	var secured bool
	var conn net.Conn = &Stream{Stream: stream}

	var cmd Command
	for {
		err := cmd.Recv(conn, s.powDifficulty)
		if err != nil {
			log.Println("ERROR: READING COMMAND:", err)
			return
		}

		switch cmd.Action {
		case ActionNoise:
			conn, err = s.handleNoise(&cmd, conn)
			if err != nil {
				log.Println("ERROR: NOISE COMMAND:", err)
				return
			}
			secured = true
		case ActionConnectInternal:
			err = s.handleConnectInternal(&cmd, conn, secured)
			if err != nil {
				log.Println("ERROR: CONNECT INTERNAL:", err)
				return
			}
		case ActionConnectExternal:
			if !secured {
				log.Println("ERROR: NOISE NOT SET")
				return
			}
			// TODO: Connect to PROTOCOL IP:PORT
			break
		default:
			log.Println("ERROR: UNKNOWN COMMAND:", cmd.Action.String())
			return
		}
	}
}

func New(cfg Config) (s *Service, err error) {
	s = &Service{
		powDifficulty: cfg.PowDifficulty,
		id:            cfg.Host.ID(),
		host:          cfg.Host,
		dht:           cfg.DHT,
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
