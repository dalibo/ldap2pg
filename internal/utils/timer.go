package utils

import (
	"time"

	"golang.org/x/exp/slog"
)

type Timer struct {
	Count int
	Total time.Duration
}

type Timeable func()

func (t *Timer) TimeIt(fn Timeable) {
	start := time.Now()
	t.Count++
	defer func() {
		duration := time.Since(start)
		slog.Debug("Took.", "duration", duration)
		t.Total += duration
	}()

	fn()
}
