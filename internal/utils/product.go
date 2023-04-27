package utils

func Product[T any](lists ...[]T) <-chan []T {
	ch := make(chan []T)
	go func() {
		defer close(ch)
		if 0 == len(lists) {
			return
		}

		indices := make([]int, len(lists))
		combination := make([]T, len(lists))
		for i, list := range lists {
			if 0 == len(list) {
				// Multiplying by empty breaks everything.
				return
			}
			combination[i] = list[0]
		}

		clone := make([]T, len(combination))
		copy(clone, combination)
		ch <- clone

		last := len(lists) - 1
		for { // Loop until we have looped the first list and last list together.

			// Each iteration, we loop lists from right to left to
			// increment the position in the list. We loop on
			// previous list only if the previous is exhausted.
			for i := last; i >= -1; i-- {
				if -1 == i {
					// We have rolled over all lists. Stop here.
					return
				}

				list := lists[i]
				// First increment. Index 0 is already sent by
				// combination 0 or by previous rollover.
				indices[i]++
				if indices[i] == len(list) {
					// (0, 1, 1) -> (0, 2, 0)
					// Reset position on this list.
					indices[i] = 0
				}

				combination[i] = list[indices[i]]

				if 0 < indices[i] {
					// Break (and yield a combination) only if we are the left-most list, the one that didn't rollover.

					// (0, 1, 1) -> (0, 1, 2)
					// OR
					// (0, 1, 0) -> (0, 2, 0) if last list had rollover.
					break
				}
			}

			clone := make([]T, len(combination))
			copy(clone, combination)
			ch <- clone
		}
	}()
	return ch
}
