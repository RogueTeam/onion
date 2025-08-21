package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/libp2p/go-libp2p/core/crypto"
)

type Listener struct {
	privKey crypto.PrivKey

	replicas      int
	circuitLength int
	onion         *onion.Onion
	database      *database.Database

	running     bool
	connections chan Connection

	circuitsMutex sync.Mutex
	circuits      []*onion.Circuit
}

type Connection struct {
	Conn  net.Conn
	Error error
}

var _ net.Listener = (*Listener)(nil)

func (l *Listener) constructCircuit(ctx context.Context) (c *onion.Circuit, err error) {
	circuitPeers, err := l.database.Circuit(database.Circuit{Length: l.circuitLength})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve circuit peers: %w", err)
	}

	circuit, err := l.onion.Circuit(ctx, circuitPeers)
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit: %w", err)
	}

	return circuit, nil
}

func (l *Listener) replicaJob() {
	defer func() {
		recover() // prevents deadly panic from channel write
	}()

	ctx := context.TODO()

	circuit, err := l.constructCircuit(ctx)
	if err != nil {
		log.Printf("failed to construct circuit: %v", err)
		return
	}
	defer circuit.Close()

	l.circuitsMutex.Lock()
	l.circuits = append(l.circuits, circuit)
	l.circuitsMutex.Unlock()

	listener, err := circuit.Bind(ctx, l.privKey)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	defer listener.Close()

	for l.running {
		var connection Connection
		connection.Conn, connection.Error = listener.Accept()
		l.connections <- connection
	}
}

func (l *Listener) replicaWorker() {
	for l.running {
		// Respawn the jobs if we are running
		l.replicaJob()
	}
}

func (l *Listener) setup() {
	for range l.replicas {
		go l.replicaWorker()
	}
}

func (l *Listener) Accept() (conn net.Conn, err error) {
	defer func() {
		err2, ok := recover().(error)
		if ok && err2 != nil {
			if err == nil {
				err = err2
			} else {
				err = fmt.Errorf("%w -> %w", err, err2)
			}
		}
	}()

	connection := <-l.connections
	return connection.Conn, connection.Error
}

func (l *Listener) Close() (err error) {
	l.running = false
	close(l.connections)

	l.circuitsMutex.Lock()
	for _, c := range l.circuits {
		c.Close()
	}
	l.circuitsMutex.Unlock()
	return nil
}

func (l *Listener) Addr() (addr net.Addr) {
	return nil
}
