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
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
)

type Src struct {
	io.ReadWriteCloser
}

func (s *Src) Read(b []byte) (n int, err error) {
	n, err = s.ReadWriteCloser.Read(b)
	os.Stdout.Write([]byte("++++++++++READ()\n"))
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("----------READ()\n"))
	return n, err
}

func (s *Src) Write(b []byte) (n int, err error) {
	n, err = s.ReadWriteCloser.Write(b)
	os.Stdout.Write([]byte("++++++++++WRITE()\n"))
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("----------WRITE()\n"))
	return n, err
}

type Stream struct {
	io.ReadWriteCloser
}

var _ net.Conn = (*Stream)(nil)

func (s *Stream) SetDeadline(t time.Time) (err error)      { return nil }
func (s *Stream) SetReadDeadline(t time.Time) (err error)  { return nil }
func (s *Stream) SetWriteDeadline(t time.Time) (err error) { return nil }

func (s *Stream) LocalAddr() (addr net.Addr) {
	return nil
}

func (s *Stream) RemoteAddr() (addr net.Addr) {
	return nil
}

const ProtocolId protocol.ID = "/my-service/0.0.1"

func main() {
	ident, err := identity.LoadIdentity("server")
	if err != nil {
		log.Fatal(err)
	}

	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/8888/quic-v1"),
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

	host.SetStreamHandler(ProtocolId, func(s network.Stream) {
		defer s.Close()

		t, err := noise.New(s.Protocol(), ident, nil)
		if err != nil {
			log.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.TODO(), time.Hour)
		defer cancel()

		sStream := Stream{ReadWriteCloser: &Src{s}}
		conn, err := t.SecureOutbound(ctx, &sStream, s.Conn().RemotePeer())
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()
		log.Println("Waiting for input")
		go io.Copy(conn, os.Stdin)
		io.Copy(os.Stdout, conn)

	})

	time.Sleep(time.Hour)
}
