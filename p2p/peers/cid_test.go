package peers_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/RogueTeam/onion/p2p/peers"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func Test_IdentityCidFromData(t *testing.T) {
	type Test struct {
		Name  string
		Value fmt.Stringer
	}
	tests := []Test{
		{Name: "peer.ID", Value: utils.Must(peer.IDFromP2PAddr(multiaddr.StringCast("/ip4/10.20.30.106/udp/44275/quic-v1/p2p/12D3KooWMxEh6cm7rMwfC6wayxyyUbRrDRFhD6SgbmfzdZhqVvCS")))},
		{Name: "Byte slice", Value: bytes.NewBuffer([]byte("\x00\x01"))},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assertions := assert.New(t)

			c := peers.IdentityCidFromData(test.Value.String())
			x := peers.PayloadFromIdentityCid[string](c)

			assertions.Equal(test.Value.String(), x, "not equal")
		})
	}
}
