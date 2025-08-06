package command

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/RogueTeam/onion/crypto"
	"github.com/RogueTeam/onion/net/compressedtunnel"
	"github.com/RogueTeam/onion/pow/hashcash"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	DefaultSaltLength = 64
)

type (
	Action      uint8
	Compression uint8
	Noise       struct {
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
		Address peer.ID `json:"address"`
	}
	Data struct {
		Settings *Settings `msgpack:",omitempty"`
		Noise    *Noise    `msgpack:",omitempty"`
		Extend   *Extend   `msgpack:",omitempty"`
		External *External `msgpack:",omitempty"`
		Bind     *Bind     `msgpack:",omitempty"`
		Dial     *Dial     `msgpack:",omitempty"`
	}
	Command struct {
		Action   Action
		Hashcash string
		Data     Data
	}
	Settings struct {
		OutsideMode   bool
		PoWDifficulty uint64
	}
)

const (
	// Send the connection settings
	ActionSettings Action = iota
	// Upgrade connection to noise channel
	ActionNoise
	// Connects to other peer in the onion network
	ActionExtend
	// Connects to a remote service
	ActionExternal
	// Bind a hidden service
	ActionBind
	// Dials to a hidden service
	ActionDial
)

func (a Action) String() (s string) {
	switch a {
	case ActionNoise:
		return "noise"
	case ActionExtend:
		return "extend"
	case ActionExternal:
		return "external"
	default:
		return fmt.Sprintf("<invalid>:%d", a)
	}
}

func (cmd *Command) Recv(r io.Reader, settings *Settings) (err error) {
	var msg compressedtunnel.Msg
	err = msg.Recv(r)
	if err != nil {
		return fmt.Errorf("failed to receive raw msg: %w", err)
	}

	*cmd = Command{}
	err = msgpack.NewDecoder(bytes.NewReader(msg.Data)).Decode(&cmd)
	if err != nil {
		return fmt.Errorf("failed to decode msgpack: %w", err)
	}

	payload, err := msgpack.Marshal(cmd.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data into payload: %w", err)
	}

	err = hashcash.VerifyWithDifficultyAndPayload(hashcash.DefaultHashAlgorithm(), cmd.Hashcash, settings.PoWDifficulty, hex.EncodeToString(payload))
	if err != nil {
		return fmt.Errorf("failed to verify hashcash: %w", err)
	}
	return nil
}

func (cmd *Command) Send(w io.Writer, settings *Settings) (err error) {
	// Prepare Command
	payload, err := msgpack.Marshal(cmd.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	cmd.Hashcash, err = hashcash.New(ctx, hashcash.DefaultHashAlgorithm(), settings.PoWDifficulty, crypto.String(DefaultSaltLength), hex.EncodeToString(payload))
	if err != nil {
		return fmt.Errorf("failed to calculate hashcash: %w", err)
	}

	// Prepare buffer to send

	cmdBytes, err := msgpack.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	// Send msg
	err = compressedtunnel.SendSingle(w, cmdBytes)
	if err != nil {
		return fmt.Errorf("failed to send msg: %w", err)
	}
	return nil
}
