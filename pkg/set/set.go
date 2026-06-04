package set

import (
	"iter"
	"maps"
)

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](members ...T) Set[T] {
	s := Set[T]{m: make(map[T]struct{}, len(members))}
	for _, m := range members {
		s.m[m] = struct{}{}
	}
	return s
}

func (s Set[T]) Add(members ...T) {
	if s.m == nil {
		s.m = make(map[T]struct{})
	}
	for _, m := range members {
		s.m[m] = struct{}{}
	}
}

func (s Set[T]) Len() int {
	return len(s.m)
}

func (s Set[T]) Contains(member T) bool {
	if s.m == nil {
		return false
	}
	_, ok := s.m[member]
	return ok
}

func (s Set[T]) Members() iter.Seq[T] {
	return maps.Keys(s.m)
}

func (s Set[T]) Delete(member T) {
	delete(s.m, member)
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := NewSet[T]()

	for e := range other.Members() {
		if s.Contains(e) {
			result.Add(e)
		}
	}

	return result
}
