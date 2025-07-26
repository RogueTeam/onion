package onion

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/RogueTeam/onion/utils"
	"github.com/klauspost/compress/gzip"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type Config struct {
	Host host.Host
	DHT  *dht.IpfsDHT
}

type Service struct {
	id      peer.ID
	privKey crypto.PrivKey
	noise   *noise.Transport
	host    host.Host
	dht     *dht.IpfsDHT
}

type (
	Action      uint8
	Compression uint8
	Command     struct {
		Action      Action
		Compression Compression
		Size        int64
		Payload     []byte
	}
)

const (
	// Upgrade connection to noise channel
	ActionNoise Action = iota
	// Connects to other peer in the onion network
	ActionConnectInternal
	// Connects to a remote service
	ActionConnectExternal
)

func (a Action) String() (s string) {
	switch a {
	case ActionNoise:
		return "noise"
	default:
		return fmt.Sprintf("<invalid>:%d", a)
	}
}

const (
	CompressionNone Compression = iota
	CompressionGzip
)

func (c Compression) String() (s string) {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionGzip:
		return "gzip"
	default:
		return fmt.Sprintf("<invalid>:%d", c)
	}
}

func PayloadFromCommand[T any](cmd *Command) (payload Payload[T], err error) {
	err = json.Unmarshal(cmd.Payload, &payload)
	if err != nil {
		return payload, fmt.Errorf("failed to decode json: %w", err)
	}
	return payload, nil
}

func (c *Command) FromReader(r io.Reader) (err error) {
	err = binary.Read(r, binary.BigEndian, &c.Action)
	if err != nil {
		return fmt.Errorf("failed to read action: %w", err)
	}

	err = binary.Read(r, binary.BigEndian, &c.Compression)
	if err != nil {
		return fmt.Errorf("failed to read compression: %w", err)
	}

	err = binary.Read(r, binary.BigEndian, &c.Size)
	if err != nil {
		return fmt.Errorf("failed to read size: %w", err)
	}

	if c.Size == 0 {
		return nil
	}

	rawPayload := make([]byte, c.Size)
	n, err := r.Read(rawPayload)
	if err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	if n != int(c.Size) {
		return fmt.Errorf("expecting payload of length %d but got length %d", c.Size, n)
	}

	switch c.Compression {
	case CompressionNone:
		c.Payload = rawPayload
	case CompressionGzip:
		r, err := gzip.NewReader(bytes.NewReader(rawPayload))
		if err != nil {
			return fmt.Errorf("failed to prepare gzip reader: %w", err)
		}
		defer r.Close()

		c.Payload, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("failed to deflate gzip: %w", err)
		}
	default:
		return fmt.Errorf("unknown compression: %s", c.Compression.String())
	}
	return nil
}

const ProtocolId protocol.ID = "/onionp2p"

type (
	Noise struct {
		PeerId peer.ID `json:"peerId"`
	}
)

// Upgrades the connection to use a noise channel
// If succeed output net.Conn is secured by encryption tunnel
func (s *Service) handleNoise(cmd *Command, conn net.Conn) (secured net.Conn, err error) {
	payload, err := PayloadFromCommand[Noise](cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to extract payload: %w", err)
	}

	err = payload.Verify()
	if err != nil {
		return nil, fmt.Errorf("failed to verify payload: %w", err)
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	secured, err = s.noise.SecureInbound(ctx, conn, payload.Data.PeerId)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade connection: %w", err)
	}

	return secured, nil
}

// Handles the stream
// On any error the stream is closed
func (s *Service) StreamHandler(stream network.Stream) {
	defer stream.Close()

	var secured bool
	var conn net.Conn = &Stream{Stream: stream}

	var cmd Command
	for {
		err := cmd.FromReader(conn)
		if err != nil {
			log.Println("ERROR: READING COMMAND:", err)
			return
		}

		switch cmd.Action {
		case ActionNoise:
			conn, err = s.handleNoise(&cmd, conn)
			if err != nil {
				log.Println("ERROR: NOISE COMMAND:", err)
				return
			}
			secured = true
		case ActionConnectInternal:
			if !secured {
				log.Println("ERROR: NOISE NOT SET")
				return
			}
		// TODO: Connect to onion enabled peer
		case ActionConnectExternal:
			if !secured {
				log.Println("ERROR: NOISE NOT SET")
				return
			}
			// TODO: Connect to PROTOCOL IP:PORT
		default:
			log.Println("ERROR: UNKNOWN COMMAND:", cmd.Action.String())
			return
		}
	}
}

func New(cfg Config) (s *Service, err error) {
	s = &Service{
		id:      cfg.Host.ID(),
		privKey: cfg.Host.Peerstore().PrivKey(s.host.ID()),
		host:    cfg.Host,
		dht:     cfg.DHT,
	}

	s.noise, err = noise.New(ProtocolId, s.privKey, []upgrader.StreamMuxer{{ID: ProtocolId, Muxer: yamux.DefaultTransport}})
	if err != nil {
		return nil, fmt.Errorf("failed to prepare noise transport: %w", err)
	}

	// Register stream handler
	cfg.Host.SetStreamHandler(ProtocolId, s.StreamHandler)
	return s, nil
}
