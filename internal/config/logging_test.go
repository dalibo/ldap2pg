package config_test

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/utils"
	"github.com/lmittmann/tint"
	"golang.org/x/exp/slog"
)

func ExampleSetLoggingHandler() {
	config.SetLoggingHandler(slog.LevelDebug, true)
	slog.Debug("Lorem ipsum dolor sit amet.", "version", utils.Version)
	slog.Info("Consectetur adipiscing elit.", "vivamus", "ut accumsan elit", "maecenas", 4.23)
	slog.Debug("Tristique nulla ac nisl dignissim.")
	slog.Debug("Eu feugiat velit dapibus. Curabitur faucibus accumsan purus.")
	slog.Error("Quisque et posuere libero.", tint.Err(fmt.Errorf("pouet")))
	// Output:
}
