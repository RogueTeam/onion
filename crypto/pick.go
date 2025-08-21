package crypto

// Randomly picks an element of the slice
func Pick[T []S, S any](s T) (v S) {
	idx := random().IntN(len(s))
	return s[idx]
}
