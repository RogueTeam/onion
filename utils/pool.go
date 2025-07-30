package utils

import "sync"

type Pool[T any] struct {
	p sync.Pool
}

func NewPool[T any]() (p *Pool[T]) {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any { return new(T) },
		},
	}
}

func (p *Pool[T]) Get() (v *T) {
	return p.p.Get().(*T)
}

func (p *Pool[T]) Put(v *T) {
	p.p.Put(v)
}
