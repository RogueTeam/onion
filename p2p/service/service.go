package service

import (
	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/onion"
)

type Service struct {
	onion    *onion.Onion
	database *database.Database
}

type Config struct {
	// Number of replicas to receive the traffic from
	Replicas int
	// Length of the circuit for each replica
	CircuitLength int
	// Onion service to use
	Onion *onion.Onion
}

func New(cfg Config) (svc *Service, err error) {
	return svc, nil
}
