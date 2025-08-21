package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/RogueTeam/onion/crypto"
	"github.com/RogueTeam/onion/p2p/database"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/elazarl/goproxy"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Easy implementation of an HTTP proxy over the onion protocol
type Proxy struct {
	proxy         *goproxy.ProxyHttpServer
	circuitLength int
	onion         *onion.Onion
	database      *database.Database
}

// Simple random function this should do some more complex checking
func (p *Proxy) constructCircuit(ctx context.Context) (circuit *onion.Circuit, err error) {
	circuitPeers, err := p.database.Circuit(database.Circuit{
		Length:     p.circuitLength,
		LastIsExit: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit peers: %w", err)
	}

	log.Println("[*] Constructing circuit:", circuitPeers)
	circuit, err = p.onion.Circuit(ctx, circuitPeers)
	if err != nil {
		return nil, fmt.Errorf("failed to construct circuit: %w", err)
	}

	return circuit, nil
}

func (p *Proxy) DialHiddenService(ctx context.Context, circuit *onion.Circuit, addr peer.ID) (conn net.Conn, err error) {
	log.Println("[*] Connecting to hidden service")

	cId := peer.ToCid(addr)
	log.Println("Searching for cid:", cId)
	candidates, err := circuit.HiddenDHT(ctx, cId)
	if err != nil {
		return nil, fmt.Errorf("failed to find hidden service candidates: %w", err)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates found for address: %s", addr)
	}

	extend := crypto.Pick(candidates)
	err = circuit.Extend(ctx, extend.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to extend circuit to candidate: %w", err)
	}

	hiddenService, err := circuit.Dial(ctx, cId)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hidden service: %w", err)
	}
	conn, err = hiddenService.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open hidden service connection: %w", err)
	}
	return conn, nil
}

func (p *Proxy) DialExternal(ctx context.Context, circuit *onion.Circuit, host, port string) (conn net.Conn, err error) {
	log.Println("[*] Connecting to public service")
	exitNodes, err := circuit.HiddenDHT(ctx, onion.ExitNodeP2PCid)
	if err != nil {
		return nil, fmt.Errorf("failed to find exit nodes: %w", err)
	}
	if len(exitNodes) == 0 {
		return nil, errors.New("no exit nodes found")
	}

	exitNode := crypto.Pick(exitNodes)
	err = circuit.Extend(ctx, exitNode.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to extend exit nodes: %w", err)
	}

	var maddr multiaddr.Multiaddr
	asAddr, _ := netip.ParseAddr(host)
	if asAddr.Is4() {
		maddr, _ = multiaddr.NewMultiaddr("/ip4/" + host + "/tcp/" + port)
	} else if asAddr.Is6() {
		maddr, _ = multiaddr.NewMultiaddr("/ip6/" + host + "/tcp/" + port)
	} else {
		maddr, _ = multiaddr.NewMultiaddr("/dnsaddr/" + host + "/tcp/" + port)
	}

	conn, err = circuit.External(ctx, maddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial external service: %w", err)
	}
	return conn, nil
}

func (p *Proxy) DialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	log.Println("[*] Connecting to", addr)
	circuit, err := p.constructCircuit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to construct circuit: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}

		if circuit != nil {
			circuit.Close()
		}
	}()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get host:port from addr: %w", err)
	}

	if strings.HasSuffix(host, ".libonion") {
		rawAddr := strings.TrimSuffix(host, ".libonion")
		addrAsPeerId, err := peer.Decode(rawAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peer id: %w", err)
		}

		return p.DialHiddenService(ctx, circuit, addrAsPeerId)
	}

	return p.DialExternal(ctx, circuit, host, port)
}

type Config struct {
	CircuitLength int
	Onion         *onion.Onion
	Database      *database.Database
}

func New(cfg Config) (p *Proxy) {
	p = &Proxy{
		proxy:         goproxy.NewProxyHttpServer(),
		circuitLength: cfg.CircuitLength,
		onion:         cfg.Onion,
		database:      cfg.Database,
	}

	p.proxy.Tr = &http.Transport{
		DialContext: p.DialContext,
	}
	return p
}

func (p *Proxy) Serve(l net.Listener) (err error) {
	err = http.Serve(l, p.proxy)
	if err != nil {
		return fmt.Errorf("failed to serve proxy: %w", err)
	}
	return nil
}
