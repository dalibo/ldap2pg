package ldap2pg

import (
	"math"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Action   CommandAction
	LogLevel zapcore.Level
}

func LoadConfig() (self Config, err error) {
	self = Config{
		Action: RunAction,
	}

	Logger.Debug("Loading Flag values.")
	flagValues := loadFlags()
	if flagValues.ShowHelp {
		self.Action = ShowHelpAction
		return
	}
	if flagValues.ShowVersion {
		self.Action = ShowVersionAction
		return
	}
	self.LoadFlags(flagValues)

	return self, nil
}

var levels []zapcore.Level = []zapcore.Level{
	zapcore.DebugLevel,
	zapcore.InfoLevel,
	zapcore.WarnLevel,
	zapcore.ErrorLevel,
	zapcore.FatalLevel,
}

func (self *Config) LoadFlags(values FlagValues) {
	verbosity := 1 - values.Verbose + values.Quiet
	verbosity = int(math.Max(0, float64(verbosity)))
	verbosity = int(math.Min(float64(verbosity), float64(len(levels)-1)))
	self.LogLevel = levels[verbosity]
}

type CommandAction int

const (
	RunAction CommandAction = iota
	ShowHelpAction
	ShowVersionAction
)

type FlagValues struct {
	Verbose     int
	Quiet       int
	Dry         bool
	ShowHelp    bool
	ShowVersion bool
	ConfigFile  string
}

func loadFlags() FlagValues {
	values := FlagValues{}
	flag.StringVarP(&values.ConfigFile, "config", "c", "", "Path to YAML configuration file.")
	flag.BoolVarP(&values.ShowHelp, "help", "?", false, "Show this help message and exit.")
	flag.BoolVarP(&values.ShowVersion, "version", "V", false, "Show version and exit.")
	flag.CountVarP(&values.Verbose, "verbose", "v", "Increase log verbosity.")
	flag.CountVarP(&values.Quiet, "quiet", "q", "Increase log verbosity.")
	flag.BoolVarP(&values.Dry, "dry", "n", true, "Don't touch Postgres, just print what to do.")
	flag.BoolVarP(&values.Dry, "real", "N", false, "Real mode, apply changes to Postgres instance.")
	flag.Parse()
	return values
}

func ShowHelp() {
	flag.Usage()
}
