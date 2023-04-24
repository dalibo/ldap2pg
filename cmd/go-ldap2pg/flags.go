package main

import (
	"math"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

func SetupConfig() {
	viper.SetDefault("color", isatty.IsTerminal(os.Stderr.Fd()))
	_ = viper.BindEnv("color")
	pflag.Bool("color", viper.GetBool("color"), "Force color output.")

	viper.SetDefault("config", "")
	_ = viper.BindEnv("config", "LDAPG2PG_CONFIG")
	pflag.StringP("config", "c", "", "Path to YAML configuration file. Use - for stdin.")

	viper.SetDefault("dry", true)
	_ = viper.BindEnv("dry", "DRY")
	pflag.BoolP("dry", "n", true, "Don't touch Postgres, just print what to do.")

	viper.SetDefault("help", false)
	pflag.BoolP("help", "?", true, "Show this help message and exit.")

	viper.SetDefault("version", false)
	pflag.BoolP("version", "V", true, "Show version and exit.")

	viper.SetDefault("quiet", 0)
	pflag.CountP("quiet", "q", "Increase log verbosity.")
	viper.SetDefault("verbose", 0)
	pflag.CountP("verbose", "v", "Increase log verbosity.")

	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
}

// Holds flags/env values to control the execution of ldap2pg.
type Controller struct {
	Color    bool
	Config   string
	Dry      bool
	Quiet    int
	Verbose  int
	LogLevel slog.Level
}

var levels []slog.Level = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	slog.LevelWarn,
	slog.LevelError,
}

func UnmarshalController() (controller Controller, err error) {
	err = viper.Unmarshal(&controller)
	// Default log level is INFO, which index is 1.
	levelIndex := 1 - viper.GetInt("verbose") + viper.GetInt("quiet")
	levelIndex = int(math.Max(0, float64(levelIndex)))
	levelIndex = int(math.Min(float64(levelIndex), float64(len(levels)-1)))
	controller.LogLevel = levels[levelIndex]
	return controller, err
}
