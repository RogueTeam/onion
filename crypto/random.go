package crypto

import (
	"crypto/rand"

	mathrand "math/rand/v2"
)

var random = mathrand.New(mathrand.NewChaCha8(
	func() (x [32]byte) {
		rand.Read(x[:])
		return x
	}(),
))
