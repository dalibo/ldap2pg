package utils

import (
	"time"
)

type StopWatch struct {
	Count int
	Total time.Duration
}

type Timeable func()

func (t *StopWatch) TimeIt(fn Timeable) (duration time.Duration) {
	start := time.Now()
	t.Count++
	defer func() {
		duration = time.Since(start)
		t.Total += duration
	}()

	fn()
	return
}
