package service

import "github.com/RogueTeam/onion/p2p/onion"

type Service struct {
	Onion *onion.Onion
}

type Config struct {
	Replicas      int
	CircuitLength int
	Onion         *onion.Onion
}

func New(cfg Config) (svc *Service, err error) {
	return svc, nil
}
