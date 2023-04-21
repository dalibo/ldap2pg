package config_test

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/utils"
	"github.com/lmittmann/tint"
	"golang.org/x/exp/slog"
)

func ExampleSetLoggingHandler() {
	colors := []bool{false, true}
	for _, color := range colors {
		config.SetLoggingHandler(slog.LevelDebug, color)
		slog.Debug("Lorem ipsum dolor sit amet.", "version", utils.Version)
		slog.Info("Consectetur adipiscing elit.", "vivamus", "ut accumsan elit", "maecenas", 4.23)
		slog.Debug("Tristique nulla ac nisl dignissim.")
		slog.Debug("Eu feugiat velit dapibus. Curabitur faucibus accumsan purus.", tint.Err(nil))
		slog.Warn("Mauris placerat molestie tempor.", "err", nil)
		slog.Error("Quisque et posuere libero.", "err", fmt.Errorf("pouet"))
	}
	// Output:
}
