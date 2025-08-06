package onion

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/RogueTeam/onion/p2p/onion/command"
	"github.com/RogueTeam/onion/utils"
	"github.com/hashicorp/yamux"
	"github.com/libp2p/go-libp2p/core/crypto"
)

// Handle the bind of a hidden service
func (c *Connection) Bind(cmd *command.Command) (err error) {
	if !c.Secured {
		return errors.New("connection not secured")
	}
	if cmd.Data.Bind == nil {
		return errors.New("bind not passed")
	}

	// Prepare public key ==================================
	rawPub, err := hex.DecodeString(cmd.Data.Bind.HexPublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode publickey: %w", err)
	}

	pub, err := crypto.UnmarshalPublicKey(rawPub)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	hiddenAddress, err := HiddenAddressFromPubKey(pub)
	if err != nil {
		return fmt.Errorf("failed to convert public key to hidden address: %w", err)
	}

	// Prepare signature ===================================
	sig, err := hex.DecodeString(cmd.Data.Bind.HexSignature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Validate signature ==================================
	valid, err := pub.Verify([]byte(hiddenAddress), sig)
	if err != nil {
		return fmt.Errorf("failed to verify publickey signature: %w", err)
	}

	if !valid {
		return errors.New("invalid signature")
	}

	ctx, cancel := utils.NewContext()
	defer cancel()

	cid, err := CidFromData(hiddenAddress)
	if err != nil {
		return fmt.Errorf("failed to create cid from pub hash: %w", err)
	}

	err = c.DHT.Provide(ctx, cid, true)
	if err != nil {
		return fmt.Errorf("failed to advertise cid: %w", err)
	}

	// Accept connections ==================================
	session, err := yamux.Client(c.Conn, yamux.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to upgrade to yamux: %w", err)
	}
	defer session.Close()

	c.HiddenServices.Store(hiddenAddress, session)
	defer c.HiddenServices.Delete(hiddenAddress)

	// Wait until caller closes. This will prevent corruption of the pipeline
	<-session.CloseChan()

	return nil
}
