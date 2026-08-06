package model

type Maybe[T any] struct {
	Val   T
	Valid bool
}

func MaybeValid[T any](v T) Maybe[T] {
	return Maybe[T]{Val: v, Valid: true}
}
