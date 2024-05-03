package internal_test

import (
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/lmittmann/tint"
)

func ExampleSetLoggingHandler() {
	colors := []bool{false, true}
	for _, color := range colors {
		internal.SetLoggingHandler(slog.LevelDebug, color)
		slog.Debug("Lorem ipsum dolor sit amet.", "version", "v1.0")
		slog.Info("Consectetur adipiscing elit.", "vivamus", "ut accumsan elit", "maecenas", 4.23)
		slog.Debug("Tristique nulla ac nisl dignissim.")
		slog.Debug("Eu feugiat velit dapibus. Curabitur faucibus accumsan purus.", tint.Err(nil))
		slog.Warn("Mauris placerat molestie tempor.", "err", nil)
		slog.Error("Quisque et posuere libero.", "err", fmt.Errorf("pouet"))
	}
	// Output:
}
