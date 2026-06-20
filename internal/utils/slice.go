package utils

func Map[T any, R any](t []T, mapFunc func(T) R) []R {
	r := make([]R, len(t))

	for i, e := range t {
		r[i] = mapFunc(e)
	}

	return r
}
