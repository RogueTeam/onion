package node

import (
	"context"
	"fmt"
	"os"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/RogueTeam/onion/p2p/onion"
	"github.com/libp2p/go-libp2p"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const (
	ExampleFlag = "example-config"
	ConfigFlag  = "config"
)

var Command = cli.Command{
	Name:        "node",
	Description: "Run in node mode. Permits connecting to the hidden network by opening a local http proxy.",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: ExampleFlag, Usage: "Write an example configuration to the passed route in the --config flag"},
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
		} else if config.FirewalledMode {
			// TODO: Find relays
			options = append(options, libp2p.NoListenAddrs, libp2p.EnableRelay())
		}

		host, err := libp2p.New(options...)
		if err != nil {
			return fmt.Errorf("failed to preapre host: %w", err)
		}

		onionCfg := onion.
			DefaultConfig().
			WithHost(host)
		svc, err := onion.New(onionCfg)
		if err != nil {
			return fmt.Errorf("failed to prepare service: %w", err)
		}

		return nil
	},
}
