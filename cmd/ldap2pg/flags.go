package main

import (
	"fmt"
	"log/slog"
	"math"
	"os"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/lithammer/dedent"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [OPTIONS]\n\n", os.Args[0])
		pflag.PrintDefaults()
		os.Stderr.Write([]byte(dedent.Dedent(`

		By default, ldap2pg runs in dry mode.
		ldap2pg requires a configuration file to describe LDAP searches and mappings.
		See https://ldap2pg.readthedocs.io/en/latest/ for further details.
		`)))
	}
}

func setupViper() {
	viper.SetDefault("check", false)
	pflag.Bool("check", viper.GetBool("check"), "Check mode: exits with 1 if Postgres instance is unsynchronized.")

	viper.SetDefault("color", defaultColor())
	_ = viper.BindEnv("color")
	pflag.Bool("color", viper.GetBool("color"), "Force color output.")

	viper.SetDefault("config", "")
	_ = viper.BindEnv("config", "LDAPG2PG_CONFIG")
	pflag.StringP("config", "c", "", "Path to YAML configuration file. Use - for stdin.")

	viper.SetDefault("real", false)
	_ = viper.BindEnv("real")
	pflag.BoolP("real", "R", viper.GetBool("real"), "Real mode. Apply changes to Postgres instance.")

	viper.SetDefault("skip-privileges", false)
	_ = viper.BindEnv("skip-privileges")
	pflag.BoolP("skip-privileges", "P", viper.GetBool("skip-privileges"), "Turn off privilege synchronisation.")

	viper.SetDefault("help", false)
	pflag.BoolP("help", "?", true, "Show this help message and exit.")

	viper.SetDefault("version", false)
	pflag.BoolP("version", "V", true, "Show version and exit.")

	viper.SetDefault("quiet", 0)
	pflag.CountP("quiet", "q", "Decrease log verbosity.")
	viper.SetDefault("verbose", 0)
	pflag.CountP("verbose", "v", "Increase log verbosity.")
	viper.SetDefault("verbosity", "")
	_ = viper.BindEnv("verbosity", "LDAP2PG_VERBOSITY")

	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
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
	SkipPrivileges bool `mapstructure:"skip-privileges"`
	Quiet          int
	Verbose        int
	Verbosity      string
	LogLevel       slog.Level
	PostgresWatch  perf.StopWatch
	LdapWatch      perf.StopWatch
}

var levels = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	internal.LevelChange,
	slog.LevelWarn,
	slog.LevelError,
}

func unmarshalController() (controller Controller, err error) {
	err = viper.Unmarshal(&controller)
	verbosity := viper.GetString("verbosity")
	var level slog.LevelVar
	switch verbosity {
	case "":
		// Default log level is INFO, which index is 1.
		levelIndex := 1 - viper.GetInt("verbose") + viper.GetInt("quiet")
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
			slog.Warn("Bad verbosity.", "source", "env", "value", verbosity)
		}
	}
	return controller, err
}
