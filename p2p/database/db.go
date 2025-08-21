package database

import (
	"log"
	"slices"
	"sync"
	"time"

	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/utils"
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
	slices.DeleteFunc(peers, func(e *onion.Peer) bool {
		return e.Info.ID == d.onion.ID
	})
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

func (d *Database) All() (peers []*onion.Peer) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return slices.Clone(d.peers)
}
