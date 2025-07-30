package set

import (
	"fmt"
	"strings"
)

type Set[T comparable] map[T]struct{}

func New[T comparable](vs ...T) (s Set[T]) {
	s = make(Set[T], len(vs))

	for _, v := range vs {
		s[v] = struct{}{}
	}

	return s
}

func (s Set[T]) Clear() {
	for v := range s {
		delete(s, v)
	}
}

func (s Set[T]) String() string {
	return fmt.Sprintf("{%s}", s.Join(", "))
}

func (s Set[T]) Join(sep string) string {
	var builder strings.Builder

	var index int
	for v := range s {
		if index == 0 {
			fmt.Fprintf(&builder, "%v", v)
		} else {
			fmt.Fprintf(&builder, "%s%v", sep, v)
		}
		index++
	}

	return builder.String()
}

func (s Set[T]) Slice() (r []T) {
	r = make([]T, 0, len(s))

	for v := range s {
		r = append(r, v)
	}

	return r
}

func (s Set[T]) Add(vs ...T) {
	for _, v := range vs {
		s[v] = struct{}{}
	}
}

func (s Set[T]) AddSet(o Set[T]) {
	for v := range o {
		s[v] = struct{}{}
	}
}

func (s Set[T]) Del(vs ...T) {
	for _, v := range vs {
		delete(s, v)
	}
}

func (s Set[T]) Union(o Set[T]) (r Set[T]) {
	r = make(Set[T], len(s))

	for v := range s {
		r[v] = struct{}{}
	}

	for v := range o {
		r[v] = struct{}{}
	}

	return r
}

func (s Set[T]) Has(v T) (found bool) {
	_, found = s[v]

	return found
}

func (s Set[T]) Intersection(o Set[T]) (r Set[T]) {
	r = make(Set[T], len(s))

	for v := range o {
		if _, found := s[v]; found {
			r[v] = struct{}{}
		}
	}

	return r
}

func (s Set[T]) Difference(o Set[T]) (r Set[T]) {
	r = make(Set[T], len(s))

	for v := range s {
		if _, found := o[v]; !found {
			r[v] = struct{}{}
		}
	}

	return r
}

func (s Set[T]) Subset(o Set[T]) (isSubset bool) {
	for v := range s {
		if _, found := o[v]; !found {
			return false
		}
	}
	return true
}
