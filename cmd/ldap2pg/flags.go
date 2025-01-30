package main

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/lithammer/dedent"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
)

var k = koanf.New(".")

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [OPTIONS] [dbname]\n\n", os.Args[0])
		pflag.PrintDefaults()
		os.Stderr.Write([]byte(dedent.Dedent(`

		Optional argument dbname is alternatively the database name or a conninfo string or an URI.
		See man psql(1) for more information.

		By default, ldap2pg runs in dry mode.
		ldap2pg requires a configuration file to describe LDAP searches and mappings.
		See https://ldap2pg.readthedocs.io/en/latest/ for further details.
		`)))
	}
}

func loadEnvAndFlags() {
	// env.Provider does not return error.
	_ = k.Load(env.Provider("LDAP2PG_", k.Delim(), func(s string) string {
		slog.Debug("Reading env.", "var", s)
		s = strings.TrimPrefix(s, "LDAP2PG_")
		s = strings.ToLower(s)
		return s
	}), nil)

	// Actually, we don't need to use k.* to set default from environment
	// because we pass k to posflag.ProviderWithFlag so that posflag provider
	// checks whether parameter is set in environment.
	pflag.Bool("check", false, "Check mode: exits with 1 if Postgres instance is unsynchronized.")
	pflag.Bool("color", defaultColor(), "Force color output.")
	pflag.StringP("config", "c", k.String("config"), "Path to YAML configuration file. Use - for stdin.")
	pflag.StringP("directory", "C", "", "Path to directory containing configuration files.")
	pflag.BoolP("real", "R", k.Bool("real"), "Real mode. Apply changes to Postgres instance.")
	pflag.BoolP("skip-privileges", "P", k.Bool("skipprivileges"), "Turn off privilege synchronisation.")
	pflag.BoolP("help", "?", false, "Show this help message and exit.")
	pflag.BoolP("version", "V", false, "Show version and exit.")
	pflag.CountP("quiet", "q", "Decrease log verbosity.")
	pflag.CountP("verbose", "v", "Increase log verbosity.")
	pflag.StringP("ldappassword-file", "y", "", "Path to LDAP password file.")
	pflag.Parse()

	// posflag.Provider does not return error.
	_ = k.Load(posflag.ProviderWithFlag(pflag.CommandLine, k.Delim(), k, func(f *pflag.Flag) (string, interface{}) {
		// remove hyphen from e.g. skip-privileges.
		key := strings.ReplaceAll(f.Name, "-", "")
		return key, posflag.FlagVal(pflag.CommandLine, f)
	}), nil)
}

func defaultColor() bool {
	plain := os.Getenv("NO_COLOR")
	if plain != "" {
		return false
	}
	return isatty.IsTerminal(os.Stderr.Fd())
}

// Controller holds flags/env values controlling the execution of ldap2pg.
type Controller struct {
	Check          bool
	Color          bool
	Config         string
	Real           bool
	SkipPrivileges bool
	Quiet          int
	Verbose        int
	Verbosity      string
	LogLevel       slog.Level
	Directory      string
	Dsn            string
}

// Finalize logs the end of ldap2pg execution and determine exit code.
func (controller Controller) Finalize(start time.Time, roles, grants, queries int) int {
	logAttrs := []interface{}{
		"searches", ldap.Watch.Count,
		"roles", roles,
		"queries", queries, // Don't use Watch.Count for dry run case.
	}
	if !controller.SkipPrivileges {
		logAttrs = append(logAttrs,
			"grants", grants,
		)
	}
	if queries > 0 {
		slog.Info("Comparison complete.", logAttrs...)
		if !controller.Real {
			slog.Info("Use --real option to apply changes.")
		}
	} else {
		slog.Info("Nothing to do.", logAttrs...)
	}

	vmPeak := perf.ReadVMPeak()
	elapsed := time.Since(start)
	slog.Info(
		"Done.",
		"elapsed", elapsed,
		"mempeak", perf.FormatBytes(vmPeak),
		"ldap", ldap.Watch.Total,
		"inspect", inspect.Watch.Total,
		"sync", postgres.Watch.Total,
	)

	if controller.Check && queries > 0 {
		return 1
	}
	return 0
}

var levels = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	internal.LevelChange,
	slog.LevelWarn,
	slog.LevelError,
}

func unmarshalController() (controller Controller, err error) {
	err = k.Unmarshal("", &controller)
	verbosity := k.String("verbosity")
	var level slog.LevelVar
	switch verbosity {
	case "":
		// Default log level is INFO, which index is 1.
		levelIndex := 1 - k.Int("verbose") + k.Int("quiet")
		levelIndex = int(math.Max(0, float64(levelIndex)))
		levelIndex = int(math.Min(float64(levelIndex), float64(len(levels)-1)))
		controller.LogLevel = levels[levelIndex]
	case "CHANGE":
		controller.LogLevel = internal.LevelChange
	default:
		err := level.UnmarshalText([]byte(verbosity))
		if err == nil {
			controller.LogLevel = level.Level()
		} else {
			slog.Error("Bad verbosity.", "source", "env", "value", verbosity)
		}
	}
	args := pflag.Args()
	if len(args) > 0 {
		controller.Dsn = args[0]
	}
	return controller, err
}
