package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"

	. "github.com/dalibo/ldap2pg/internal" //nolint:revive
)

func main() {
	// Split error management from business logic. This allows defer to
	// apply before calling os.Exit. Also, deduplicate fatal error logging.
	// Simply return an error and main will handle this case.
	err := run()
	if err != nil {
		slog.Error("Fatal error.", "error", err)
		os.Exit(1)
	}
}

func run() (err error) {
	start := time.Now()

	err = SetupLogging()
	if err != nil {
		return
	}

	config := NewConfig()
	err = config.Load()
	if err != nil {
		return
	}
	switch config.Action {
	case ShowHelpAction:
		ShowHelp()
		return
	case ShowVersionAction:
		showVersion()
		return
	case RunAction:
	}

	SetLoggingHandler(config.LogLevel)
	slog.Info("Starting ldap2pg",
		"commit", ShortRevision,
		"version", Version,
		"runtime", runtime.Version())

	slog.Info("Using YAML configuration file.",
		"path", config.ConfigFile,
		"version", config.Version)

	if config.Dry {
		slog.Warn("Dry run. Postgres instance will be untouched.")
	} else {
		slog.Info("Running in real mode. Postgres instance will modified.")
	}

	instance, err := PostgresInspect(config)
	if err != nil {
		return
	}

	wanted, err := ComputeWanted(config)
	if err != nil {
		return
	}

	prefix := ""
	if config.Dry {
		prefix = "Would "
	}
	for query := range wanted.Diff(instance) {
		slog.Info(prefix+query.Description, query.LogArgs...)
	}

	elapsed := time.Since(start)
	slog.Info("Comparison complete.", "elapsed", elapsed)
	return
}

func showVersion() {
	fmt.Printf("go-ldap2pg %s\n", Version)

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	modmap := make(map[string]string)
	for _, mod := range bi.Deps {
		modmap[mod.Path] = mod.Version
	}
	modules := []string{
		"github.com/jackc/pgx/v4",
		"github.com/go-ldap/ldap/v3",
		"gopkg.in/yaml.v3",
	}
	for _, mod := range modules {
		fmt.Printf("%s %s\n", mod, modmap[mod])
	}

	fmt.Printf("%s %s %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
