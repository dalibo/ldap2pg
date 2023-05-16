package lists

import (
	mapset "github.com/deckarep/golang-set/v2"
)

func IterateToSet[T comparable](ch <-chan any, target *mapset.Set[T]) error {
	for item := range ch {
		err, _ := item.(error)
		if err != nil {
			return err
		}
		(*target).Add(item.(T))
	}
	return nil
}

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
