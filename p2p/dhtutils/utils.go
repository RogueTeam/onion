package dhtutils

import (
	"context"
	"fmt"
	"log"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

func WaitForBootstrap(ctx context.Context, h host.Host, d *dht.IpfsDHT) (err error) {
	log.Println("[*] Waiting for bootstraping")
	defer log.Println("[+] Bootstrap completed")
	err = d.Bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap dht: %w", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return err
			}
			return nil
		case <-ticker.C:
			log.Println("[*] Current peerstore state:", h.Peerstore().Peers().Len())

			log.Println("[*] Current routing table state:", len(d.RoutingTable().ListPeers()))
			if h.Peerstore().Peers().Len() > 0 && len(d.RoutingTable().ListPeers()) > 0 {
				return nil
			}
		}
	}
}
