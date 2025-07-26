package onion

import (
	"crypto/sha512"
	"fmt"

	"github.com/RogueTeam/onion/pow/hashcash"
	"github.com/libp2p/go-libp2p/core/peer"
)

var DefaultHashAlgorithm = sha512.New512_256

type Payload[T any] struct {
	Sender   peer.ID `json:"sender"`
	Hashcash string  `json:"hashcash"`
	Data     T       `json:"data"`
}

func (p *Payload[T]) Verify() (err error) {
	// Verify Hashcash
	err = hashcash.Verify(DefaultHashAlgorithm(), p.Hashcash)
	if err != nil {
		return fmt.Errorf("failed to verify hash: %w", err)
	}

	// TODO: Verify signature
	return nil
}
