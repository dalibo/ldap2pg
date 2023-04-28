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
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// Split error management from business logic. This allows defer to
	// apply before calling os.Exit. Also, deduplicate fatal error logging.
	// Simply return an error and main will handle this case.
	err := ldap2pg()
	if err != nil {
		slog.Error("Fatal error.", "err", err)
		os.Exit(1)
	}
}

func ldap2pg() (err error) {
	// Bootstrap logging first to log in setup.
	config.SetLoggingHandler(slog.LevelInfo, isatty.IsTerminal(os.Stderr.Fd()))
	SetupConfig()
	if viper.GetBool("help") {
		pflag.Usage()
		return
	} else if viper.GetBool("version") {
		showVersion()
		return
	}
	err = sync()
	return
}

func sync() (err error) {
	start := time.Now()

	controller, err := UnmarshalController()
	if err != nil {
		return
	}

	config.SetLoggingHandler(controller.LogLevel, controller.Color)
	slog.Info("Starting ldap2pg",
		"commit", utils.ShortRevision,
		"version", utils.Version,
		"runtime", runtime.Version())

	configPath := config.FindConfigFile(controller.Config)
	slog.Info("Using YAML configuration file.", "path", configPath)
	c, err := config.Load(configPath)
	if err != nil {
		return
	}

	instance, err := states.PostgresInspect(c)
	if err != nil {
		return
	}

	wanted, err := states.ComputeWanted(c, instance.RolesBlacklist)
	if err != nil {
		return
	}

	if controller.Real {
		slog.Info("Real mode. Postgres instance will modified.")
	} else {
		slog.Warn("Dry run. Postgres instance will be untouched.")
	}

	count, err := wanted.Sync(&controller.PostgresTimer, controller.Real, instance)

	vmPeak := utils.ReadVMPeak()
	elapsed := time.Since(start)
	logAttrs := []interface{}{
		"queries", count,
		"elapsed", elapsed,
		"mempeak", utils.FormatBytes(vmPeak),
		"postgres", controller.PostgresTimer.Total,
	}
	if count > 0 {
		slog.Info("Comparison complete.", logAttrs...)
	} else {
		slog.Info("Nothing to do.", logAttrs...)
	}

	if err != nil {
		return
	}

	if controller.Check && count > 0 {
		os.Exit(1)
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
		"github.com/jackc/pgx/v5",
		"github.com/go-ldap/ldap/v3",
		"gopkg.in/yaml.v3",
	}
	for _, mod := range modules {
		fmt.Printf("%s %s\n", mod, modmap[mod])
	}

	fmt.Printf("%s %s %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
