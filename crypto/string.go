package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

// Generates a crypto string
func String(length int) (s string) {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}
