package utils

import "sync"

type Map[K, T any] struct {
	sync.Map
}

func (m *Map[K, T]) Load(k K) (v T, found bool) {
	rawV, found := m.Map.Load(k)
	return rawV.(T), found
}

func (m *Map[K, T]) Store(k K, v T) {
	m.Map.Store(k, v)
}
