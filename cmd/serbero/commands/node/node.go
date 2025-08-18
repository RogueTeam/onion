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
	"github.com/libp2p/go-libp2p/core/peer"
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
			log.Println("-", addr.Encapsulate(multiaddr.StringCast("/p2p/"+host.ID().String())))
		}

		dhtOptions := []dht.Option{
			dht.Datastore(datastore.NewMapDatastore()),
		}
		if config.Bootstrap != nil {
			var infos []peer.AddrInfo
			if len(config.Bootstrap.Hosts) > 0 {
				log.Println("[*] Using defined bootstrap hosts")
				for _, addr := range config.Bootstrap.Hosts {
					info, _ := peer.AddrInfoFromP2pAddr(addr)
					if info != nil {
						log.Printf("[+] Custom Bootstrap host: %v", info)
						infos = append(infos, *info)
					}
				}
			}
			if config.Bootstrap.Defaults {
				log.Println("[*] Using default bootstrap hosts")
				infos = append(infos, dht.GetDefaultBootstrapPeerAddrInfos()...)
			}
			if len(infos) > 0 {
				dhtOptions = append(dhtOptions, dht.BootstrapPeers(infos...))
			}
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

		log.Println("[*] Preparing Service")
		onionCfg := onion.
			DefaultConfig().
			WithHost(host).
			WithDHT(hostDht).
			WithTTL(config.TTL).
			WithBootstrap(config.Bootstrap != nil && config.Bootstrap.Wait).
			WithExitNode(config.ExitNode)
		svc, err := onion.New(onionCfg)
		if err != nil {
			return fmt.Errorf("failed to prepare service: %w", err)
		}

		log.Println("[*] Preparing proxy")
		if config.Proxy != nil {
			listener, err := manet.Listen(config.Proxy.ListenAddress)
			if err != nil {
				return fmt.Errorf("failed to listen proxy: %w", err)
			}
			defer listener.Close()

			log.Println("[*] Proxy Listening at:", listener.Multiaddr())
			go func() {
				p := proxy.New(proxy.Config{
					CircuitLength:        config.Proxy.CircuitLength,
					Onion:                svc,
					PeersRefreshInterval: time.Minute,
				})
				err := p.Serve(manet.NetListener(listener))
				if err != nil {
					log.Fatal(err)
				}
			}()
		} else {
			log.Println("[*] Proxy disabled")
		}

		log.Println("[*] Booting services")
		for _, service := range config.Services {
			log.Printf("[*] Booting: %s (%s) listening at: %s", service.Name, service.IdentityLocation, service.LocalAddress)
			// TODO: Implement me!

		}

		time.Sleep(time.Hour)

		return nil
	},
}
