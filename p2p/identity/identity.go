package identity

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func NewKey() (privKey crypto.PrivKey, err error) {
	privKey, _, err = crypto.GenerateEd25519Key(rand.Reader)
	return privKey, err
}

func LoadIdentity(location string) (privKey crypto.PrivKey, err error) {
	if _, err := os.Stat(location); err == nil {
		log.Println("[*] Loading existing Key Pair")
		contents, err := os.ReadFile(location)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		privKey, err = crypto.UnmarshalPrivateKey(contents)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal key: %w", err)
		}
		return privKey, nil
	} else if os.IsNotExist(err) {
		log.Println("[*] Generating Key Pair")
		privKey, err = NewKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}

		contents, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key: %w", err)
		}

		err = os.WriteFile(location, contents, 0o660)
		if err != nil {
			return nil, fmt.Errorf("failed to save private key: %w", err)
		}

		return privKey, nil
	} else {
		log.Println("[!] Failed to load Key Pair")
		return nil, fmt.Errorf("failed to load key: %w", err)
	}
}
