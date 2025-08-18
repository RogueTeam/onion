package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/RogueTeam/onion/proxy"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFlag = "config"
)

var Command = cli.Command{
	Name:        "node",
	Description: "Run in node mode. Permits connecting to the hidden network by opening a local http proxy.",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: ConfigFlag, Required: true, Usage: "Configuration YAML file to load", Value: "config.yaml"},
	},
	Action: func(ctx context.Context, cmd *cli.Command) (err error) {
		filename := cmd.String(ConfigFlag)

		contents, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var config Config
		err = yaml.Unmarshal(contents, &config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal configuration: %w", err)
		}

		// Prepare the service ======================================
		privKey, err := identity.LoadIdentity(config.IdentityLocation)
		if err != nil {
			return fmt.Errorf("failed to load identity from file: %w", err)
		}

		var options []libp2p.Option = []libp2p.Option{libp2p.Identity(privKey)}
		if config.HiddenMode {
			options = append(options, libp2p.NoListenAddrs)
		} else {
			options = append(options, libp2p.ListenAddrs(config.ListenAddresses...))
		}

		if !config.HiddenMode && len(config.AdvertiseAddresses) != 0 {
			options = append(options, libp2p.AddrsFactory(func(m []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				return config.AdvertiseAddresses
			}))
		}

		host, err := libp2p.New(options...)
		if err != nil {
			return fmt.Errorf("failed to preapre host: %w", err)
		}
		defer host.Close()

		log.Println("[*] Listening at:")
		for _, addr := range host.Addrs() {
			log.Println("-", addr)
		}

		dhtOptions := []dht.Option{
			dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...),
			dht.Datastore(datastore.NewMapDatastore()),
		}
		if config.HiddenMode {
			dhtOptions = append(dhtOptions, dht.Mode(dht.ModeClient))
		} else {
			dhtOptions = append(dhtOptions, dht.Mode(dht.ModeServer))
		}
		hostDht, err := dht.New(context.TODO(), host, dhtOptions...)
		if err != nil {
			return fmt.Errorf("failed to setup dht: %w", err)
		}
		defer hostDht.Close()

		onionCfg := onion.
			DefaultConfig().
			WithHost(host).
			WithDHT(hostDht).
			WithTTL(config.TTL).
			WithBootstrap(config.Bootstrap).
			WithExitNode(config.ExitNode)
		svc, err := onion.New(onionCfg)
		if err != nil {
			return fmt.Errorf("failed to prepare service: %w", err)
		}

		if config.Proxy != nil {
			listener, err := manet.Listen(config.Proxy.ListenAddress)
			if err != nil {
				return fmt.Errorf("failed to listen proxy: %w", err)
			}
			defer listener.Close()

			log.Println("[*] Proxy Listening at:", config.Proxy.ListenAddress)
			go func() {
				p := proxy.Proxy{
					CircuitLength:        config.Proxy.CircuitLength,
					Service:              svc,
					Listener:             manet.NetListener(listener),
					PeersRefreshInterval: time.Minute,
				}
				err := p.Serve()
				if err != nil {
					log.Fatal(err)
				}
			}()
		}

		time.Sleep(time.Hour)

		return nil
	},
}
