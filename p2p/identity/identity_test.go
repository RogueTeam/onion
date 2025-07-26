package identity_test

import (
	"os"
	"testing"

	"github.com/RogueTeam/onion/p2p/identity"
	"github.com/stretchr/testify/assert"
)

func Test_NewKey(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		key1, err := identity.NewKey()
		assertions.Nil(err, "failed to generate key 1", err)

		key2, err := identity.NewKey()
		assertions.Nil(err, "failed to generate key 2", err)

		assertions.False(key1.Equals(key2), "keys should be different", err)
	})
}

func Test_LoadIdentity(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		temp, err := os.CreateTemp("", "*")
		assertions.Nil(err, "failed to create temporary file")
		temp.Close()
		os.RemoveAll(temp.Name())
		defer os.RemoveAll(temp.Name())

		key1, err := identity.LoadIdentity(temp.Name())
		assertions.Nil(err, "failed to generate key 1", err)

		key2, err := identity.LoadIdentity(temp.Name())
		assertions.Nil(err, "failed to generate key 2", err)

		key3, err := identity.LoadIdentity(temp.Name() + "-2")
		assertions.Nil(err, "failed to generate key 2", err)
		defer os.RemoveAll(temp.Name() + "-2")

		assertions.True(key1.Equals(key2), "keys should be equal: %w", err)
		assertions.False(key1.Equals(key3), "keys should be different: %w", err)
	})
}
