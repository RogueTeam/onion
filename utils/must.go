package utils

func Must[T any](v T, err error) (v2 T) {
	if err != nil {
		panic(err)
	}
	return v
}
