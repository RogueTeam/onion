package proxy

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	mathrand "math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/set"
	"github.com/RogueTeam/onion/utils"
	"github.com/elazarl/goproxy"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

var random = mathrand.New(mathrand.NewChaCha8(
	func() (x [32]byte) {
		rand.Read(x[:])
		return x
	}(),
))

// Easy implementation of an HTTP proxy over the onion protocol
type Proxy struct {
	proxy                *goproxy.ProxyHttpServer
	circuitLength        int
	onion                *onion.Onion
	peersRefreshInterval time.Duration

	once sync.Once

	peersMutex sync.Mutex
	allPeers   []*onion.Peer
}

func (p *Proxy) refreshPeers() {
	p.once.Do(func() {
		ticker := time.NewTicker(p.peersRefreshInterval)
		defer ticker.Stop()
		for {
			func() {
				p.peersMutex.Lock()
				defer p.peersMutex.Unlock()

				log.Println("[*] Refreshing peer list")
				defer log.Println("[+] Refreshed peer list")
				ctx, cancel := utils.NewContext()
				defer cancel()

				allPeers, err := p.onion.ListPeers(ctx)
				if err != nil {
					log.Println("[!] Failed to refresh peer list:", err)
				}
				slices.DeleteFunc(allPeers, func(e *onion.Peer) bool {
					return e.Info.ID == p.onion.ID
				})
				p.allPeers = allPeers
			}()
			<-ticker.C
		}
	})
}

// Simple random function this should do some more complex checking
func (p *Proxy) constructCircuit(ctx context.Context) (circuit *onion.Circuit, err error) {
	p.peersMutex.Lock()
	allPeers := slices.Clone(p.allPeers)
	p.peersMutex.Unlock()
	random.Shuffle(len(allPeers), func(i, j int) {
		allPeers[i] = allPeers[j]
	})

	peers := set.New[peer.ID]()
	for _, peer := range allPeers[:min(p.circuitLength, len(allPeers))] {
		if peer == nil {
			continue
		}
		peers.Add(peer.Info.ID)
	}
	log.Println("[*] Peers", peers)

	log.Println("[*] Constructing circuit")
	circuit, err = p.onion.Circuit(ctx, peers.Slice())
	if err != nil {
		return nil, fmt.Errorf("failed to construct circuit: %w", err)
	}

	return circuit, nil
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
		log.Println("[*] Connecting to hidden service")

		rawAddr := strings.TrimSuffix(host, ".libonion")
		peerId, err := peer.Decode(rawAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peer id: %w", err)
		}
		log.Println("Raw address:", peerId)
		cId := peer.ToCid(peerId)
		log.Println("Searching for cid:", cId)
		candidates, err := circuit.HiddenDHT(ctx, cId)
		if err != nil {
			return nil, fmt.Errorf("failed to find hidden service candidates: %w", err)
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("no candidates found for address: %s", host)
		}

		random.Shuffle(len(candidates), func(i, j int) { candidates[i] = candidates[j] })

		err = circuit.Extend(ctx, candidates[0].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to extend circuit to candidate: %w", err)
		}

		hiddenService, err := circuit.Dial(ctx, cId)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to hidden service: %w", err)
		}
		conn, err := hiddenService.Open(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to open hidden service connection: %w", err)
		}
		return conn, nil
	}

	log.Println("[*] Connecting to public service")
	exitNodes, err := circuit.HiddenDHT(ctx, onion.ExitNodeP2PCid)
	if err != nil {
		return nil, fmt.Errorf("failed to find exit nodes: %w", err)
	}
	if len(exitNodes) == 0 {
		return nil, errors.New("no exit nodes found")
	}

	random.Shuffle(len(exitNodes), func(i, j int) { exitNodes[i] = exitNodes[j] })
	err = circuit.Extend(ctx, exitNodes[0].ID)
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

type Config struct {
	CircuitLength        int
	Onion                *onion.Onion
	PeersRefreshInterval time.Duration
}

func New(cfg Config) (p *Proxy) {
	p = &Proxy{
		proxy:                goproxy.NewProxyHttpServer(),
		circuitLength:        cfg.CircuitLength,
		onion:                cfg.Onion,
		peersRefreshInterval: cfg.PeersRefreshInterval,
	}

	p.proxy.Tr = &http.Transport{
		DialContext: p.DialContext,
	}
	return p
}

func (p *Proxy) Serve(l net.Listener) (err error) {
	go p.refreshPeers()

	err = http.Serve(l, p.proxy)
	if err != nil {
		return fmt.Errorf("failed to serve proxy: %w", err)
	}
	return nil
}
