package ldap2pg

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"

	"github.com/kelseyhightower/envconfig"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Action   CommandAction
	LogLevel zapcore.Level
	Ldap     struct {
		URI      string
		BindDn   string
		Password string
	}
	ConfigFile string
}

func LoadConfig() (config Config, err error) {
	config = Config{
		Action: RunAction,
		// Default to current LogLevel.
		LogLevel: LogLevel.Level(),
	}

	Logger.Debug("Loading Flag values.")
	flagValues := loadFlags()
	if flagValues.ShowHelp {
		config.Action = ShowHelpAction
		return
	}
	if flagValues.ShowVersion {
		config.Action = ShowVersionAction
		return
	}
	config.LoadFlags(flagValues)

	Logger.Debug("Loading Environment values.")
	var envValues EnvValues
	envconfig.MustProcess("", &envValues)
	config.LoadEnv(envValues)

	if config.ConfigFile == "" {
		config.ConfigFile, err = config.FindConfigFile()
		if err != nil {
			return config, err
		}
	}

	err = config.LoadYaml()
	if err != nil {
		return config, err
	}

	return config, nil
}

func (config *Config) FindConfigFile() (configpath string, err error) {
	Logger.Debugw("Searching configuration file in standard locations.")
	me, _ := user.Current()
	candidates := []string{
		"./ldap2pg.yml",
		"./ldap2pg.yaml",
		path.Join(me.HomeDir, "/.config/ldap2pg.yml"),
		path.Join(me.HomeDir, "/.config/ldap2pg.yaml"),
		"/etc/ldap2pg.yml",
		"/etc/ldap2pg.yaml",
	}

	for _, candidate := range candidates {
		_, err := os.Stat(candidate)
		if err == nil {
			Logger.Debugw("Found configuration file.", "path", candidate)
			return candidate, nil
		}
		Logger.Debugw("Ignoring configuration file.", "path", candidate, "error", err)
	}

	return "", fmt.Errorf("No configuration file found")
}

var levels []zapcore.Level = []zapcore.Level{
	zapcore.DebugLevel,
	zapcore.InfoLevel,
	zapcore.WarnLevel,
	zapcore.ErrorLevel,
	zapcore.FatalLevel,
}

func (config *Config) LoadFlags(values FlagValues) {
	change := 0 - values.Verbose + values.Quiet
	if change != 0 {
		var levelIndex int
		for i, level := range levels {
			if level == config.LogLevel {
				levelIndex = i
				break
			}
		}

		levelIndex = levelIndex + change
		levelIndex = int(math.Max(0, float64(levelIndex)))
		levelIndex = int(math.Min(float64(levelIndex), float64(len(levels)-1)))
		config.LogLevel = levels[levelIndex]
		Logger.Debugw("Setting log level.", "source", "flags", "level", config.LogLevel)
	}

	if values.ConfigFile != "" {
		Logger.Debugw("Setting config file.", "source", "flags", "path", values.ConfigFile)
		config.ConfigFile = values.ConfigFile
	}
}

func (config *Config) LoadEnv(values EnvValues) {
	Logger.Debugw("Setting LDAPURI.", "source", "env", "value", values.LdapURI)
	config.Ldap.URI = values.LdapURI
	Logger.Debugw("Setting LDAPBINDDN.", "source", "env", "value", values.LdapBindDn)
	config.Ldap.BindDn = values.LdapBindDn
	Logger.Debugw("Setting LDAPPASSWORD.", "source", "env")
	config.Ldap.Password = values.LdapPassword

	if config.ConfigFile == "" && values.ConfigFile != "" {
		Logger.Debugw("Setting config file.", "source", "env", "path", values.ConfigFile)
		config.ConfigFile = values.ConfigFile
	}
}

type EnvValues struct {
	LdapURI        string `envconfig:"LDAPURI"`
	LdapBindDn     string `envconfig:"LDAPBINDDN"`
	LdapPassword   string `envconfig:"LDAPPASSWORD"`
	LdapTLSReqcert string `envconfig:"LDAPTLS_REQCERT"`
	Dry            bool   `envconfig:"DRY" default:"true"`
	ConfigFile     string `envconfig:"LDAP2PG_CONFIG"`
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

func (config *Config) LoadYaml() (err error) {
	fo, err := os.Open(config.ConfigFile)
	if err != nil {
		return
	}
	var y YamlConfig
	dec := yaml.NewDecoder(fo)
	err = dec.Decode(&y)
	if err != nil {
		return
	}

	switch y.(type) {
	case map[string]interface{}:
		Logger.Debugw("YAML is a map", "value", y)
	case []interface{}:
		Logger.Debugw("YAML is a list", "value", y)
	}
	return
}

type YamlConfig interface{}
