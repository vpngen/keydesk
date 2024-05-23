package filter

import (
	"cmp"
	"fmt"
)

type Interface[T any] interface {
	Filter(values []T) []T
}

type Fn[T any] func(values []T) []T

func (fn Fn[T]) Pipe(fns ...Interface[T]) Fn[T] {
	return func(values []T) []T {
		for _, f := range fns {
			values = f.Filter(values)
		}
		return values
	}
}

func (fn Fn[T]) Filter(values []T) (ret []T) {
	return fn(values)
}

type Func[T any] func(T) bool

func (f Func[T]) And(f2 Func[T]) Func[T] {
	return func(v T) bool {
		return f(v) && f2(v)
	}
}

func (f Func[T]) Or(f2 Func[T]) Func[T] {
	return func(v T) bool {
		return f(v) || f2(v)
	}
}

func (f Func[T]) Not() Func[T] {
	return func(t T) bool {
		return !f(t)
	}
}

func (f Func[T]) IfOrTrue(f2 Func[T]) Func[T] {
	return func(v T) bool {
		if f2(v) {
			return f(v)
		}
		return true
	}
}

func (f Func[T]) Filter(values []T) (ret []T) {
	for _, v := range values {
		if f(v) {
			ret = append(ret, v)
		}
	}
	return
}

func Filter[T any](values []T, filters ...Interface[T]) []T {
	for _, filter := range filters {
		values = filter.Filter(values)
	}
	return values
}

func Ordered[T cmp.Ordered](op string, v T) Func[T] {
	switch op {
	case "eq", "==":
		return func(t T) bool {
			return t == v
		}
	case "ne", "!=":
		return func(t T) bool {
			return t != v
		}
	case "gt", ">":
		return func(t T) bool {
			return t > v
		}
	case "lt", "<":
		return func(t T) bool {
			return t < v
		}
	case "ge", ">=":
		return func(t T) bool {
			return t >= v
		}
	case "le", "<=":
		return func(t T) bool {
			return t <= v
		}
	}
	panic(fmt.Sprintf("unknown operator %q", op))
}
