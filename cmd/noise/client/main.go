package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

const ProtocolId protocol.ID = "/my-service/0.0.1"

type Stream struct {
	network.Stream
}

var _ net.Conn = (*Stream)(nil)

func (s *Stream) LocalAddr() (addr net.Addr) {
	return nil
}

func (s *Stream) RemoteAddr() (addr net.Addr) {
	return nil
}

func main() {
	rawAddr := os.Args[1]

	remoteAddr, err := multiaddr.NewMultiaddr(rawAddr)
	if err != nil {
		log.Fatal(err)
	}
	remoteId, err := peer.IDFromP2PAddr(remoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	ident, err := identity.LoadIdentity("client")
	if err != nil {
		log.Fatal(err)
	}

	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/9999/quic-v1"),
		libp2p.Identity(ident),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer host.Close()

	maddr, err := multiaddr.NewMultiaddr("/p2p/" + host.ID().String())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[+] Id:", host.ID())
	log.Println("[+] Listening")

	for _, addr := range host.Addrs() {
		log.Println("\t-", addr.Encapsulate(maddr))
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	target := peer.AddrInfo{
		ID:    remoteId,
		Addrs: []multiaddr.Multiaddr{remoteAddr},
	}
	log.Println("[*] Connecting to:", target)
	err = host.Connect(ctx, target)
	if err != nil {
		log.Fatal(err)
	}

	stream, err := host.NewStream(ctx, remoteId, ProtocolId)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	log.Println("[*] Negotiating")
	t, err := noise.New(stream.Protocol(), ident, nil)
	if err != nil {
		log.Fatal(err)
	}

	sStream := Stream{Stream: stream}
	conn, err := t.SecureInbound(ctx, &sStream, remoteId)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Println("[*] Waiting for input")
	go io.Copy(conn, os.Stdin)
	io.Copy(os.Stdout, conn)

}
