package internal

import (
	"fmt"
	"os"

	"github.com/dalibo/ldap2pg/internal/tint"
	"github.com/mattn/go-isatty"
	"golang.org/x/exp/slog"
)

var currentLogLevel slog.Level

func SetupLogging() error {
	_, debug := os.LookupEnv("DEBUG")
	level := new(slog.LevelVar)
	if debug {
		level.Set(slog.LevelDebug)
	} else {
		// Early configuration using environment variable, to debug initialization.
		envlevel, found := os.LookupEnv("LDAP2PG_VERBOSITY")
		if found {
			err := level.UnmarshalText([]byte(envlevel))
			if err != nil {
				return fmt.Errorf("Bad LDAP2PG_VERBOSITY value: %s", envlevel)
			}
		}
	}

	SetLoggingHandler(level.Level())

	slog.Debug("Initializing ldap2pg.", "version", Version)
	return nil
}

func SetLoggingHandler(level slog.Level) {
	currentLogLevel = level
	var h slog.Handler
	if isatty.IsTerminal(os.Stderr.Fd()) {
		h = tint.Options{
			Level: level,
			LevelStrings: map[slog.Level]string{
				slog.LevelDebug: "\033[0;2mDEBUG",
				slog.LevelInfo:  "\033[0;1mINFO ",
				slog.LevelWarn:  "\033[0;1;38;5;185mINFO ",
				slog.LevelError: "\033[0;1;31mERROR",
			},
			TimeFormat: "15:04:05",
		}.NewHandler(os.Stderr)
	} else {
		h = slog.HandlerOptions{
			Level: level,
		}.NewTextHandler(os.Stderr)
	}
	slog.SetDefault(slog.New(h))
}
