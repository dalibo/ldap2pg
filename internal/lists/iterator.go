package lists

func IterateToSlice[T any](ch <-chan any, target *[]T) error {
	for item := range ch {
		err, _ := item.(error)
		if err != nil {
			return err
		}
		*target = append(*target, item.(T))
	}
	return nil
}
