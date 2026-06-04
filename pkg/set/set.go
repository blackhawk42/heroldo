// Package set provides a generic Set implementation backed by a map.
package set

import (
	"iter"
	"maps"
)

// Set is a generic set collection for comparable types.
//
// The zero value (var s Set[T]) is ready for use.
type Set[T comparable] struct {
	// m is the underlying map that stores set members as keys.
	m map[T]struct{}
}

// NewSet returns a new Set populated with the given members.
func NewSet[T comparable](members ...T) Set[T] {
	s := Set[T]{m: make(map[T]struct{}, len(members))}
	for _, m := range members {
		s.m[m] = struct{}{}
	}
	return s
}

// Add inserts one or more members into the set.
//
// Lazily allocates the underlying map on first use, so the zero value is ready for use.
func (s *Set[T]) Add(members ...T) {
	if s.m == nil {
		s.m = make(map[T]struct{})
	}
	for _, m := range members {
		s.m[m] = struct{}{}
	}
}

// Len returns the number of members in the set.
func (s Set[T]) Len() int {
	return len(s.m)
}

// Contains reports whether the given member is in the set.
func (s Set[T]) Contains(member T) bool {
	if s.m == nil {
		return false
	}
	_, ok := s.m[member]
	return ok
}

// Members returns an iterator over all members of the set.
func (s Set[T]) Members() iter.Seq[T] {
	return maps.Keys(s.m)
}

// Delete removes the given member from the set.
func (s Set[T]) Delete(member T) {
	delete(s.m, member)
}

// Intersection returns a new set containing only the members present in
// both s and other.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := NewSet[T]()

	for e := range other.Members() {
		if s.Contains(e) {
			result.Add(e)
		}
	}

	return result
}
