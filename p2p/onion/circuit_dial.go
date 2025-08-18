package onion

import (
	"context"
	"fmt"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/hashicorp/yamux"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type HiddenServiceConnection struct {
	Address cid.Cid
	Noise   *noise.Transport
	Session *yamux.Session
}

func (h *HiddenServiceConnection) Close() (err error) {
	return h.Session.Close()
}

func (h *HiddenServiceConnection) Open(ctx context.Context) (conn sec.SecureConn, err error) {
	insecure, err := h.Session.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	id, err := peer.FromCid(h.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain peer id from cid: %w", err)
	}
	conn, err = h.Noise.SecureOutbound(ctx, insecure, id)
	if err != nil {
		return nil, fmt.Errorf("failed upgrade connection: %w", err)
	}
	return conn, nil
}

// Receives the DefaultHashAlgorithm of the public key of the hidden service and returns a yamux.Session
// The yamux session can create multiple dials to the same address using the session.Open method.
// The circuit should be constructed in order to force the last node be the one advertising the service.
// If not, the connection will fail
func (c *Circuit) Dial(ctx context.Context, address cid.Cid) (hidden *HiddenServiceConnection, err error) {
	var dial = message.Message{
		Data: message.Data{
			Dial: &message.Dial{
				Address: address,
			},
		},
	}
	err = dial.Send(ctx, c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send dial: %w", err)
	}

	session, err := yamux.Server(c.Active, yamux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to negotiate connection: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		session.Close()
	}()

	hiddenIdentity, err := identity.NewKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate hidden identity: %w", err)
	}

	noiseTransport, err := noise.New(ProtocolId, hiddenIdentity, DefaultMuxerUpgrader)
	if err != nil {
		return nil, fmt.Errorf("failed upgrade noise transport: %w", err)
	}

	hidden = &HiddenServiceConnection{
		Address: address,
		Noise:   noiseTransport,
		Session: session,
	}
	return hidden, nil
}
