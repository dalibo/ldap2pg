package lists

func And[T any](s []T, fn func(T) bool) bool {
	for _, i := range s {
		if !fn(i) {
			return false
		}
	}
	return true
}
