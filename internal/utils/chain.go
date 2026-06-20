package utils

func BuildChain[T any, F ~func(T) T](items []F, target T) T {
	for i := len(items) - 1; i >= 0; i-- {
		target = items[i](target)
	}

	return target
}
