package onion

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/RogueTeam/onion/p2p/log"
	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
)

func (s *Service) handleConnectInternal(logger *log.Logger, cmd *command.Command, conn net.Conn, secured bool) (err error) {
	if !secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.ConnectInternal == nil {
		return errors.New("connect internal not passed")
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	info, err := s.dht.FindPeer(ctx, cmd.Data.ConnectInternal.PeerId)
	if err != nil {
		return fmt.Errorf("couldn't find peer: %w", err)
	}

	err = s.host.Connect(ctx, info)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	logger.Log(log.LogLevelDebug, "Connected to peer: %v", info)

	stream, err := s.host.NewStream(ctx, info.ID, ProtocolId)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	logger.Log(log.LogLevelDebug, "Piping traffic")
	defer logger.Log(log.LogLevelDebug, "Finished")
	go io.Copy(conn, stream)
	io.Copy(stream, conn)

	return nil
}
