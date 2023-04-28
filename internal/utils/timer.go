package utils

import (
	"time"

	"golang.org/x/exp/slog"
)

type Timer struct {
	Total time.Duration
}

type Timeable func()

func (t *Timer) TimeIt(fn Timeable) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		slog.Debug("Took.", "duration", duration)
		t.Total += duration
	}()

	fn()
}
