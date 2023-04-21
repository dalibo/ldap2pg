package config

import (
	"fmt"
	"os"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/lmittmann/tint"
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

	colorEnv, found := os.LookupEnv("COLOR")
	var color bool
	if found {
		color = "true" == colorEnv
	} else {
		color = isatty.IsTerminal(os.Stderr.Fd())
	}
	SetLoggingHandler(level.Level(), color)

	return nil
}

var levelStrings = map[slog.Level]string{
	slog.LevelDebug: "\033[2mDEBUG",
	slog.LevelInfo:  "\033[1mINFO ",
	slog.LevelWarn:  "\033[1;38;5;185mWARN ",
	slog.LevelError: "\033[1;31mERROR",
}

func SetLoggingHandler(level slog.Level, color bool) {
	currentLogLevel = level
	var h slog.Handler
	if color {
		h = BuildTintOptions(level).NewHandler(os.Stderr)
	} else {
		h = slog.HandlerOptions{
			Level: level,
		}.NewTextHandler(os.Stderr)
	}
	slog.SetDefault(slog.New(h))
}

func BuildTintOptions(level slog.Level) tint.Options {
	return tint.Options{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Value = slog.StringValue(levelStrings[slog.Level(a.Value.Int64())])
			case slog.MessageKey:
				// Reset color after message.
				a.Value = slog.StringValue(a.Value.String() + "\033[0m")
			default:
				if a.Value.Kind() == slog.KindAny {
					v := a.Value.Any()
					set, ok := v.(mapset.Set[string])
					if ok {
						a.Value = slog.AnyValue(set.ToSlice())
						return a
					}
					if nil == v && "err" == a.Key {
						// Drop nil error.
						a.Key = ""
						return a
					}
					// Automatic tint.Err()
					err, ok := v.(error)
					if ok {
						a = tint.Err(err)
					}
				}
			}
			return a
		},
		TimeFormat: "15:04:05",
	}
}
