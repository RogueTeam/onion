package message

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/crypto"
	"github.com/RogueTeam/onion/net/compressedtunnel"
	"github.com/RogueTeam/onion/pow/hashcash"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	DefaultSaltLength = 64
)

type (
	Settings struct {
		ExitNode      bool
		PoWDifficulty uint64
	}
	Noise struct {
		PeerPublicKey []byte `json:"peerId"`
	}
	Extend struct {
		PeerId peer.ID `json:"peerId"`
	}
	External struct {
		Address multiaddr.Multiaddr `json:"address"`
	}
	Bind struct {
		// Hex encoded public key
		HexPublicKey string `json:"publicKey"`
		// Hex encoded signature of the DefaultHashAlgorithm of the public key
		HexSignature string
	}
	Dial struct {
		// Address of the hidden service
		Address cid.Cid `json:"address"`
	}
	// HiddenDHT msg used for querying anonymously the IPFS HiddenDHT without revealing who is doing it
	HiddenDHT struct {
		Cid cid.Cid // Target Cid requested
	}
	HiddenDHTResponse struct {
		Peers []peer.AddrInfo
	}
	Data struct {
		Settings          *Settings          `msgpack:",omitempty"`
		Noise             *Noise             `msgpack:",omitempty"`
		Extend            *Extend            `msgpack:",omitempty"`
		External          *External          `msgpack:",omitempty"`
		Bind              *Bind              `msgpack:",omitempty"`
		Dial              *Dial              `msgpack:",omitempty"`
		HiddenDHT         *HiddenDHT         `msgpack:",omitempty"`
		HiddenDHTResponse *HiddenDHTResponse `msgpack:",omitempty"`
	}
	Message struct {
		Hashcash string
		Data     Data
	}
)

func (m *Message) Recv(r io.Reader, settings *Settings) (err error) {
	var compressedMsg compressedtunnel.Msg
	err = compressedMsg.Recv(r)
	if err != nil {
		return fmt.Errorf("failed to receive raw msg: %w", err)
	}

	*m = Message{}
	err = msgpack.NewDecoder(bytes.NewReader(compressedMsg.Data)).Decode(&m)
	if err != nil {
		return fmt.Errorf("failed to decode msgpack: %w", err)
	}

	payload, err := msgpack.Marshal(m.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data into payload: %w", err)
	}

	err = hashcash.VerifyWithDifficultyAndPayload(hashcash.DefaultHashAlgorithm(), m.Hashcash, settings.PoWDifficulty, hex.EncodeToString(payload))
	if err != nil {
		return fmt.Errorf("failed to verify hashcash: %w", err)
	}
	return nil
}

func (m *Message) Send(ctx context.Context, w io.Writer, settings *Settings) (err error) {
	// Prepare Msg
	payload, err := msgpack.Marshal(m.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	m.Hashcash, err = hashcash.New(ctx, hashcash.DefaultHashAlgorithm(), settings.PoWDifficulty, crypto.String(DefaultSaltLength), hex.EncodeToString(payload))
	if err != nil {
		return fmt.Errorf("failed to calculate hashcash: %w", err)
	}

	// Prepare buffer to send

	msgBytes, err := msgpack.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	// Send msg
	err = compressedtunnel.SendSingle(w, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send msg: %w", err)
	}
	return nil
}
