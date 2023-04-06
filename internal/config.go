package internal

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"

	"github.com/kelseyhightower/envconfig"
	"github.com/lithammer/dedent"
	flag "github.com/spf13/pflag"
	"golang.org/x/exp/slog"
)

type Config struct {
	Action     CommandAction
	ConfigFile string
	// Use a tristate because we load config from highest priority to lowest.
	MaybeDry int  // Tristate: 0 = undefined, -1 = false, 1 = true
	Dry      bool // Final value for app.
	LogLevel slog.Level
	Version  int
	Ldap     struct {
		URI      string
		BindDn   string
		Password string
	}
	Postgres PostgresQueries
	SyncMap  []SyncItem
}

type PostgresQueries struct {
	DatabasesQuery      InspectQuery
	ManagedRolesQuery   InspectQuery
	RolesBlacklistQuery InspectQuery
}

func NewConfig() Config {
	return Config{
		Action:   RunAction,
		LogLevel: currentLogLevel,
		Postgres: PostgresQueries{
			DatabasesQuery: InspectQuery{
				Name: "databases_query",
				Default: dedent.Dedent(`
				SELECT datname FROM pg_catalog.pg_database
				WHERE datallowconn IS TRUE ORDER BY 1;`),
			},
			ManagedRolesQuery: InspectQuery{
				Name: "managed_roles_query",
			},
			RolesBlacklistQuery: InspectQuery{
				Name: "roles_blacklist_query",
				// Inject Static value as returned by YAML
				Default: []interface{}{"pg_*", "postgres"},
			},
		},
	}
}

func (config *Config) Load() (err error) {
	slog.Debug("Loading Flag values.")
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

	slog.Debug("Loading Environment values.")
	var envValues EnvValues
	envconfig.MustProcess("", &envValues)
	config.LoadEnv(envValues)

	slog.Debug("Loading YAML configuration.")
	if config.ConfigFile == "" {
		config.ConfigFile = config.FindConfigFile()
		if config.ConfigFile == "" {
			return fmt.Errorf("No configuration file found")
		}
	}

	yamlValues, err := ReadYaml(config.ConfigFile)
	if err != nil {
		return
	}
	err = config.LoadYaml(yamlValues)
	if err != nil {
		return
	}

	config.LoadDefaults()

	config.Dry = config.MaybeDry >= 0

	return
}

func (config *Config) FindConfigFile() (configpath string) {
	slog.Debug("Searching configuration file in standard locations.")
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
			slog.Debug("Found configuration file.",
				"path", candidate)

			return candidate
		}
		slog.Debug("Ignoring configuration file.",
			"path", candidate,
			"error", err)
	}

	return ""
}

var levels []slog.Level = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	slog.LevelWarn,
	slog.LevelError,
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
		slog.Debug("Setting log level.",
			"source", "flags",
			"level", config.LogLevel.String())

	}

	if values.ConfigFile != "" {
		slog.Debug("Setting config file.",
			"source", "flags",
			"path", values.ConfigFile)

		config.ConfigFile = values.ConfigFile
	}

	config.MaybeDry = values.MaybeDry
}

func (config *Config) LoadEnv(values EnvValues) {
	if 0 == config.MaybeDry && "" != values.RawDry {
		slog.Debug("Setting DRY.", "source", "env", "value", values.RawDry)
		if values.Dry {
			config.MaybeDry = 1
		} else {
			config.MaybeDry = -1
		}
	}

	if values.LdapURI != "" {
		slog.Debug("Setting LDAPURI.",
			"source", "env",
			"value", values.LdapURI)

		config.Ldap.URI = values.LdapURI
	}

	if values.LdapBindDn != "" {
		slog.Debug("Setting LDAPBINDDN.",
			"value", values.LdapBindDn,
			"source", "env")

		config.Ldap.BindDn = values.LdapBindDn
	}

	if values.LdapPassword != "" {
		slog.Debug("Setting LDAPPASSWORD.",
			"source", "env")

		config.Ldap.Password = values.LdapPassword
	}

	if config.ConfigFile == "" && values.ConfigFile != "" {
		slog.Debug("Setting config file.",
			"source", "env",
			"path", values.ConfigFile)

		config.ConfigFile = values.ConfigFile
	}
}

type Tristate int

type EnvValues struct {
	LdapURI        string `envconfig:"LDAPURI"`
	LdapBindDn     string `envconfig:"LDAPBINDDN"`
	LdapPassword   string `envconfig:"LDAPPASSWORD"`
	LdapTLSReqcert string `envconfig:"LDAPTLS_REQCERT"`
	RawDry         string `envconfig:"DRY"` // Tri state: "" = undefined, else boolean.
	Dry            bool   `envconfig:"DRY"` // Tri state: "" = undefined, else boolean.
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
	Real        bool
	MaybeDry    int // Tristate
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
	flag.BoolVarP(&values.Dry, "dry", "n", false, "Don't touch Postgres, just print what to do.")
	flag.BoolVarP(&values.Real, "real", "N", false, "Real mode, apply changes to Postgres instance.")
	flag.Parse()

	// Apply --real or --dry only if set. --dry prevales.
	if values.Real {
		slog.Debug("Setting real mode.", "source", "flags")
		values.MaybeDry = -1
	}
	if values.Dry {
		slog.Debug("Setting dry mode.", "source", "flags")
		values.MaybeDry = 1
	}

	return values
}

func ShowHelp() {
	flag.Usage()
}

func (config *Config) LoadDefaults() {
	config.Postgres.DatabasesQuery.SetDefault()
	config.Postgres.RolesBlacklistQuery.SetDefault()
	if 0 == config.MaybeDry {
		config.Dry = true
	}
}
