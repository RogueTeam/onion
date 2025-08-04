package onion

import (
	"github.com/RogueTeam/onion/p2p/log"
	"github.com/libp2p/go-libp2p/core/network"
)

// Handles the stream
// On any error the stream is closed
func (s *Service) StreamHandler(stream network.Stream) {
	defer stream.Close()

	conn := Connection{
		Host:     s.Host,
		DHT:      s.DHT,
		Conn:     &NetConnStream{Stream: stream},
		Settings: s.Settings,
		Stream:   stream,
		Logger: log.Logger{
			PeerID: stream.Conn().RemotePeer(),
		},
		Noise:          s.Noise,
		Secured:        false,
		ExternalMode:   s.Settings.OutsideMode,
		HiddenServices: s.HiddenServices,
	}
	err := conn.Handle()
	if err != nil {
		conn.Logger.Log(log.LogLevelError, "failed to handle peer connection: %v", err)
	}
}
