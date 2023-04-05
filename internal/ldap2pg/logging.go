package ldap2pg

import (
	"fmt"
	"os"

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
	slog.SetDefault(slog.New(slog.HandlerOptions{
		Level: level,
	}.NewTextHandler(os.Stderr)))

	slog.Debug("Initializing ldap2pg.", "version", Version)
}
