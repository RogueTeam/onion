package onion

import (
	"net"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/libp2p/go-libp2p/core/network"
)

// Handles the stream
// On any error the stream is closed
func (s *Service) StreamHandler(stream network.Stream) {
	defer stream.Close()

	var logger = log.Logger{
		PeerID: stream.Conn().RemotePeer(),
	}

	// Send Settings
	var settings = command.Command{
		Action: command.ActionSettings,
		Data: command.Data{
			Settings: &s.settings,
		},
	}
	err := settings.Send(stream, DefaultSettings)
	if err != nil {
		logger.Log(log.LogLevelError, "SENDING SETTINGS: %v", err)
		return
	}
	//

	var secured bool
	var conn net.Conn = &Stream{Stream: stream}

	var cmd command.Command
	for {
		err := cmd.Recv(conn, &s.settings)
		if err != nil {
			logger.Log(log.LogLevelError, "READING COMMAND: %v", err)
			return
		}

		switch cmd.Action {
		case command.ActionNoise:
			conn, err = s.handleNoise(&cmd, conn)
			if err != nil {
				logger.Log(log.LogLevelError, "NOISE COMMAND: %v", err)
				return
			}
			secured = true
		case command.ActionConnectInternal:
			err = s.handleConnectInternal(&cmd, conn, secured)
			if err != nil {
				logger.Log(log.LogLevelError, "CONNECT INTERNAL: %v", err)
				return
			}
		case command.ActionConnectExternal:
			// TODO: Connect to PROTOCOL IP:PORT
			break
		default:
			logger.Log(log.LogLevelError, "UNKNOWN COMMAND: %v", cmd.Action.String())
			return
		}
	}
}
