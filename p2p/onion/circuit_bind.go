package onion

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/p2p/onion/message"
	"github.com/RogueTeam/onion/utils"
	"github.com/hashicorp/yamux"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type HiddenServiceListener struct {
	Noise   *noise.Transport
	PrivKey crypto.PrivKey
	Session *yamux.Session
}

func (h *HiddenServiceListener) Close() (err error) {
	return h.Session.Close()
}

// Accept hidden service connections
func (h *HiddenServiceListener) Accept() (conn io.ReadWriteCloser, err error) {
	insecure, err := h.Session.Accept()
	if err != nil {
		return nil, fmt.Errorf("failed to accept connection: %w", err)
	}

	ctx, _ := utils.NewContext()

	conn, err = h.Noise.SecureInbound(ctx, insecure, "")
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade insecure: %w", err)
	}
	return conn, nil
}

// Binds a hidden service based on a private key
func (c *Circuit) Bind(priv crypto.PrivKey) (h *HiddenServiceListener, err error) {
	hiddenAddress, err := HiddenAddressFromPrivKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to get address from private key: %w", err)
	}

	pubMarshaled, err := crypto.MarshalPublicKey(priv.GetPublic())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	sign, err := priv.Sign([]byte(hiddenAddress))
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	var bind = message.Message{
		Data: message.Data{
			Bind: &message.Bind{
				HexPublicKey: hex.EncodeToString(pubMarshaled),
				HexSignature: hex.EncodeToString(sign),
			},
		},
	}
	err = bind.Send(c.Active, c.Settings[c.Current])
	if err != nil {
		return nil, fmt.Errorf("failed to send bind: %w", err)
	}

	session, err := yamux.Server(c.Active, yamux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade yamux: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		session.Close()
	}()

	noiseTransport, err := noise.New(ProtocolId, priv, DefaultMuxerUpgrader)
	if err != nil {
		return nil, fmt.Errorf("failed to create noise tranport: %w", err)
	}

	h = &HiddenServiceListener{
		Noise:   noiseTransport,
		PrivKey: priv,
		Session: session,
	}
	return h, nil
}
