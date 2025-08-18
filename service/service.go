package service

import "github.com/RogueTeam/onion/p2p/onion"

type Service struct {
	Onion *onion.Service
}

type Config struct {
	Replicas      int
	CircuitLength int
	Onion         *onion.Service
}

func New(cfg Config) (svc *Service, err error) {
	return svc, nil
}
