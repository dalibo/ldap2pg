package internal

import (
	"fmt"
	"log/slog"
	"os"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/lmittmann/tint"
)

// Level for changes only. Aka Magnus owns level. See #219
const LevelChange slog.Level = slog.LevelInfo + 2

var CurrentLevel slog.Level = slog.LevelInfo

func SetLoggingHandler(level slog.Level, color bool) {
	var h slog.Handler
	if color {
		h = tint.NewHandler(os.Stderr, buildTintOptions(level))
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if slog.LevelKey == a.Key {
					if a.Value.Any().(slog.Level) == LevelChange {
						a.Value = slog.StringValue("CHANGE")
					}
				}
				return a
			},
		})
	}
	slog.SetDefault(slog.New(h))
	CurrentLevel = level
}

var levelStrings = map[string]string{
	// Colors from journalctl. Pad with spaces to fit 6 characters.
	"DEBUG":  "\033[2mDEBUG ",
	"INFO":   "\033[1mINFO  ",
	"INFO+2": "\033[1mCHANGE",
	"WARN":   "\033[1;38;5;185mWARN  ",
	"ERROR":  "\033[1;31mERROR ",
}

func buildTintOptions(level slog.Level) *tint.Options {
	return &tint.Options{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Value = slog.StringValue(levelStrings[a.Value.String()])
			case slog.MessageKey:
				// Reset color after message.
				a.Value = slog.StringValue(fmt.Sprintf("%-48s", a.Value.String()) + "\033[0m")
			default:
				if a.Value.Kind() != slog.KindAny {
					return a
				}
				v := a.Value.Any()
				switch v := v.(type) {
				case mapset.Set[string]:
					a.Value = slog.AnyValue(v.ToSlice())
					return a
				case error: // Automatic tint.Err()
					a = tint.Err(v)
				case nil:
					if "err" == a.Key {
						a.Key = "" // Drop nil error.
						return a
					}
				}
			}
			return a
		},
		TimeFormat: "15:04:05",
	}
}
