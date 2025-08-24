package node

import (
	"errors"
	"time"

	"github.com/multiformats/go-multiaddr"
)

type Service struct {
	Name             string              `yaml:"name,omitempty"`
	Replicas         int                 `yaml:"replicas,omitempty"`
	CircuitLength    int                 `yaml:"circuit-length"`
	LocalAddress     multiaddr.Multiaddr `yaml:"local-address"`
	IdentityLocation string              `yaml:"identity-location"`
}

type Proxy struct {
	ListenAddress multiaddr.Multiaddr `yaml:"proxy-address"`
	CircuitLength int                 `yaml:"circuit-length"`
}

type Bootstrap struct {
	Defaults bool                  `yaml:"defaults,omitempty"`
	Hosts    []multiaddr.Multiaddr `yaml:"hosts,omitempty"`
	Wait     bool                  `yaml:"wait,omitempty"`
}

type Config struct {
	// Hosted services promoted in the network
	Services []Service `yaml:"services"`
	// Address used by the host to advertise itself
	AdvertiseAddresses []multiaddr.Multiaddr `yaml:"advertise-addresses"`
	// Address to listen
	ListenAddresses []multiaddr.Multiaddr `yaml:"listen-addresses"`
	// Time to live for refreshing entries in the DHT
	TTL time.Duration `yaml:"ttl"`
	// Location of the libp2p node identity
	IdentityLocation string `yaml:"identity-location"`
	// Forces the service to not listen into any port
	// and not promote itself into the network
	HiddenMode bool `yaml:"hidden-mode"`
	// Exit node mode. Permits peers to connect outside the network.
	ExitNode bool `yaml:"exit-node"`
	// Wait for bootstraping the node
	Bootstrap *Bootstrap `yaml:"bootstrap,omitempty"`
	// HTTP Proxy address
	Proxy *Proxy `yaml:"proxy,omitempty"`
}

func (c *Config) Validate() (err error) {
	if c.IdentityLocation == "" {
		return errors.New("no identity-location file provided in the configuration")
	}
	return nil
}
