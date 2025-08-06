package node

import "errors"

type Config struct {
	// Location of the libp2p node identity
	IdentityLocation string `yaml:"identity-location"`
	// Forces the service to not listen into any port
	// and not promote itself into the network
	HiddenMode bool `yaml:"hidden-mode"`
	// Firewalled mode enables to participate in the network without exposing any public address to the internet
	// The node will search for any relay node available to use
	FirewalledMode bool `yaml:"relay-mode"`
}

func (c *Config) Validate() (err error) {
	if c.IdentityLocation == "" {
		return errors.New("no identity-location file provided in the configuration")
	}
	return nil
}
