package onion

import (
	"github.com/RogueTeam/onion/p2p/log"
	"github.com/libp2p/go-libp2p/core/network"
)

// Handles the stream
// On any error the stream is closed
func (s *Service) StreamHandler(stream network.Stream) {
	defer stream.Close()

	settings := s.Settings()
	defer s.Connections.Add(-1)

	conn := Connection{
		Host:     s.Host,
		DHT:      s.DHT,
		Conn:     &NetConnStream{Stream: stream},
		Settings: settings,
		Stream:   stream,
		Logger: log.Logger{
			PeerID: stream.Conn().RemotePeer(),
		},
		Noise:          s.Noise,
		Secured:        false,
		ExitNode:       s.ExitNode,
		HiddenServices: s.HiddenServices,
	}

	err := conn.Handle()
	if err != nil {
		conn.Logger.Log(log.LogLevelError, "failed to handle peer connection: %v", err)
	}
}
