package log

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelError LogLevel = "ERROR"
	LogLevelIfo   LogLevel = "INFO"
)

type Logger struct {
	PeerID peer.ID
}

func (l *Logger) Log(level LogLevel, msg string, vs ...any) {
	base := fmt.Sprintf("%s: %s %s", level, l.PeerID, msg)
	log.Printf(base, vs...)
}
