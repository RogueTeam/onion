package crypto

import (
	"crypto/rand"

	mathrand "math/rand/v2"
)

func random() (r *mathrand.Rand) {
	var x [32]byte
	rand.Read(x[:])
	return mathrand.New(mathrand.NewChaCha8(x))
}
