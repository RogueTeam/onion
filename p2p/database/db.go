package database

import (
	"log"
	"slices"
	"sync"
	"time"

	"github.com/RogueTeam/onion/crypto"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/set"
	"github.com/RogueTeam/onion/utils"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Peers database for local caching of remote peers
type Database struct {
	refreshInterval time.Duration
	onion           *onion.Onion

	initialized bool
	ready       chan struct{}

	running bool
	mutex   sync.Mutex
	peers   []*onion.Peer
}

func (d *Database) doRefresh() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	log.Println("[*] Refreshing peer list")
	defer log.Println("[+] Refreshed peer list")
	ctx, cancel := utils.NewContext()
	defer cancel()

	peers, err := d.onion.ListPeers(ctx)
	if err != nil {
		log.Println("[!] Failed to refresh peer list:", err)
		return
	}
	d.peers = peers

	if !d.initialized {
		d.initialized = true
		d.ready <- struct{}{}
	}
}

func (d *Database) refreshWorker() {
	d.running = true

	ticker := time.NewTicker(d.refreshInterval)
	defer ticker.Stop()

	for d.running {
		d.doRefresh()
		<-ticker.C
	}
}

func (d *Database) Close() (err error) {
	d.running = false
	return nil
}

type Config struct {
	Onion           *onion.Onion
	RefreshInterval time.Duration
}

func New(cfg Config) (db *Database) {
	db = &Database{
		refreshInterval: cfg.RefreshInterval,
		onion:           cfg.Onion,
		ready:           make(chan struct{}, 1),
	}
	go db.refreshWorker()
	<-db.ready

	return db
}

// Random shiffled peers. Self peer is not include
func (d *Database) All() (peers []*onion.Peer) {
	d.mutex.Lock()
	all := slices.Clone(d.peers)
	d.mutex.Unlock()

	crypto.Shuffle(all)
	return all
}

type Circuit struct {
	// Specifies the length of the circuit.
	// In case the length can't be satisfied depending on MandatoryLength an error will be returned
	Length int
	// Specifies if last node of the circuit should be a exit node
	LastIsExit bool
}

func (d *Database) Circuit(c Circuit) (circuitPeers []peer.ID, err error) {
	all := d.All()

	circuitPeers = make([]peer.ID, 0, c.Length)

	var added = set.New[peer.ID]()
	for index := range c.Length - 1 {
		peer := all[index]

		if added.Has(peer.Info.ID) {
			continue
		}

		if c.LastIsExit && peer.Modes.Has(onion.ExitNodeP2PCid) {
			continue
		}

		added.Add(peer.Info.ID)
		circuitPeers = append(circuitPeers, peer.Info.ID)
	}

	for _, peer := range all {
		if added.Has(peer.Info.ID) {
			continue
		}

		if !c.LastIsExit || peer.Modes.Has(onion.ExitNodeP2PCid) {
			circuitPeers = append(circuitPeers, peer.Info.ID)
			break
		}
	}
	return circuitPeers, nil
}
