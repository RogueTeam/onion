package crypto

func Shuffle[T []S, S any](s T) {
	random().Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})
}
