package main

import (
	"context"
	"log"
	"os"

	"github.com/RogueTeam/onion/cmd/serbero/commands/example"
	"github.com/RogueTeam/onion/cmd/serbero/commands/node"
	"github.com/urfave/cli/v3"
)

var app = cli.Command{
	Name:        "serbero",
	Description: "libp2p PoC hidden network. Not named after the famous dog but rather after a local hero in Southamerican History Books",
	Commands: []*cli.Command{
		&node.Command,
		&example.Command,
	},
}

func main() {
	err := app.Run(context.TODO(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
