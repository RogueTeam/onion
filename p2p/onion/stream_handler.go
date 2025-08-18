package onion

import (
	"context"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/libp2p/go-libp2p/core/network"
)

// Handles the stream
// On any error the stream is closed
func (o *Onion) StreamHandler(stream network.Stream) {
	defer stream.Close()

	settings := o.Settings()
	defer o.Connections.Add(-1)

	conn := Connection{
		Host:     o.Host,
		DHT:      o.DHT,
		Conn:     &NetConnStream{Stream: stream},
		Settings: settings,
		Stream:   stream,
		Logger: log.Logger{
			PeerID: stream.Conn().RemotePeer(),
		},
		Noise:          o.Noise,
		Secured:        false,
		ExitNode:       o.ExitNode,
		HiddenServices: o.HiddenServices,
	}

	// TODO: Add some kind of limit to the connection handling
	err := conn.Handle(context.TODO())
	if err != nil {
		conn.Logger.Log(log.LogLevelError, "failed to handle peer connection: %v", err)
	}
}
