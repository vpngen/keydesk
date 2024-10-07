package utils

type (
	Set[T comparable] map[T]struct{}
	StringSet         = Set[string]
)

func New[T comparable](values ...T) Set[T] {
	s := make(Set[T], len(values))
	s.Add(values...)
	return s
}

func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) Add(values ...T) {
	for _, value := range values {
		s[value] = struct{}{}
	}
}

func (s Set[T]) Remove(v T) {
	delete(s, v)
}

func (s Set[T]) Slice() []T {
	result := make([]T, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}

func (s Set[T]) Intersect(s2 Set[T]) Set[T] {
	result := Set[T]{}
	for k := range s {
		if s2.Contains(k) {
			result.Add(k)
		}
	}
	return result
}

func (s Set[T]) Union(s2 Set[T]) Set[T] {
	result := Set[T]{}
	for k := range s {
		result.Add(k)
	}
	for k := range s2 {
		result.Add(k)
	}
	return result
}

func (s Set[T]) Difference(s2 Set[T]) Set[T] {
	result := Set[T]{}
	for k := range s {
		if !s2.Contains(k) {
			result.Add(k)
		}
	}
	return result
}
