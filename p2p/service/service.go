package service

import (
	"net"

	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/set"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Service struct {
	replicas      int
	circuitLength int
	onion         *onion.Onion
	database      *database.Database
}

type Config struct {
	// Number of replicas to receive the traffic from
	Replicas int
	// Length of the circuit for each replica
	CircuitLength int
	// Onion service to use
	Onion *onion.Onion
	// Database for circuit construction
	Database *database.Database
}

func (s *Service) Listen(priv crypto.PrivKey) (l net.Listener, err error) {
	lr := &Listener{
		privKey:       priv,
		replicas:      s.replicas,
		circuitLength: s.circuitLength,
		onion:         s.onion,
		database:      s.database,
		running:       true,
		connections:   make(chan Connection, 1_000),
		usedPeers:     set.New[peer.ID](),
	}
	go lr.setup()
	return lr, nil
}

func New(cfg Config) (svc *Service) {
	svc = &Service{
		replicas:      cfg.Replicas,
		circuitLength: cfg.CircuitLength,
		onion:         cfg.Onion,
		database:      cfg.Database,
	}

	return svc
}
