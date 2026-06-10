package cmd

import (
	"log/slog"
	"os"

	"github.com/pkg/profile"
)

func startProfiling() stopper {
	pprof := os.Getenv("LDAP2PG_PROFILE")
	if pprof == "" {
		return nil
	}
	options := []func(*profile.Profile){profile.ProfilePath(".")}
	switch pprof {
	case "block":
		options = append(options, profile.BlockProfile)
	case "clock":
		options = append(options, profile.ClockProfile)
	case "goroutine":
		options = append(options, profile.GoroutineProfile)
	case "mem":
		options = append(options, profile.MemProfile, profile.MemProfileRate(1024))
	case "mutex":
		options = append(options, profile.MutexProfile)
	case "trace":
		options = append(options, profile.TraceProfile)
	case "cpu":
		options = append(options, profile.CPUProfile)
	default:
		slog.Warn("Unknown profile type.", "type", pprof)
	}
	return profile.Start(options...)
}

type stopper interface {
	Stop()
}
