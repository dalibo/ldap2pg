package internal

import (
	"os"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/lmittmann/tint"
	"golang.org/x/exp/slog"
)

const LevelChange slog.Level = slog.LevelInfo + 2

var levelStrings = map[slog.Level]string{
	slog.LevelDebug: "\033[2mDEBUG ",
	slog.LevelInfo:  "\033[1mINFO  ",
	// Level for changes only. Aka Magnus owns level. See #219
	LevelChange:     "\033[1mCHANGE",
	slog.LevelWarn:  "\033[1;38;5;185mWARN  ",
	slog.LevelError: "\033[1;31mERROR ",
}

func SetLoggingHandler(level slog.Level, color bool) {
	var h slog.Handler
	if color {
		h = BuildTintOptions(level).NewHandler(os.Stderr)
	} else {
		h = slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if slog.LevelKey == a.Key {
					if a.Value.Any().(slog.Level) == LevelChange {
						a.Value = slog.StringValue("CHANGE")
					}
				}
				return a
			},
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
