package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"golang.org/x/exp/slog"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/states"
	"github.com/dalibo/ldap2pg/internal/utils"
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

	c, err := config.Load()
	if err != nil {
		return
	}
	switch c.Action {
	case config.ShowHelpAction:
		config.ShowHelp()
		return
	case config.ShowVersionAction:
		showVersion()
		return
	case config.RunAction:
	}

	config.SetLoggingHandler(c.LogLevel)
	slog.Info("Starting ldap2pg",
		"commit", utils.ShortRevision,
		"version", utils.Version,
		"runtime", runtime.Version())

	slog.Info("Using YAML configuration file.",
		"path", c.ConfigFile,
		"version", c.Version)

	if c.Dry {
		slog.Warn("Dry run. Postgres instance will be untouched.")
	} else {
		slog.Info("Running in real mode. Postgres instance will modified.")
	}

	instance, err := states.PostgresInspect(c)
	if err != nil {
		return
	}

	wanted, err := states.ComputeWanted(c)
	if err != nil {
		return
	}

	count, err := wanted.Sync(c, instance)

	elapsed := time.Since(start)
	if count > 0 {
		slog.Info("Comparison complete.", "queries", count, "elapsed", elapsed)
	} else {
		slog.Info("Nothing to do.", "queries", 0, "elapsed", elapsed)
	}
	return
}

func showVersion() {
	fmt.Printf("go-ldap2pg %s\n", utils.Version)

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
