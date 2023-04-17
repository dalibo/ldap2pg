package config

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"

	"github.com/dalibo/ldap2pg/internal/utils"
	"github.com/kelseyhightower/envconfig"
	"github.com/lithammer/dedent"
	"github.com/mattn/go-isatty"
	flag "github.com/spf13/pflag"
	"golang.org/x/exp/slog"
)

type Config struct {
	Action     CommandAction
	MaybeColor Tristate
	Color      bool
	ConfigFile string
	// Use a tristate because we load config from highest priority to lowest.
	MaybeDry Tristate
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

func New() Config {
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

func Load() (Config, error) {
	// Bootstrap logging before loading config.
	err := SetupLogging()
	if err != nil {
		return Config{}, err
	}
	slog.Debug("Initializing ldap2pg.", "version", utils.Version)
	c := New()
	err = c.Load()
	return c, err
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

	config.Color = config.MaybeColor.Bool()
	config.Dry = config.MaybeDry.Bool()

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
	config.MaybeColor = values.MaybeColor
}

func (config *Config) LoadEnv(values EnvValues) {
	if !config.MaybeColor.Defined() && "" != values.RawColor {
		slog.Debug("Setting color mode.", "source", "env", "value", values.RawColor)
		config.MaybeColor.Set(values.Color)
	}

	if !config.MaybeDry.Defined() && "" != values.RawDry {
		slog.Debug("Setting DRY.", "source", "env", "value", values.RawDry)
		config.MaybeDry.Set(values.Dry)
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

func (t Tristate) Bool() bool {
	return t > 0
}

func (t Tristate) Defined() bool {
	return t != 0
}

func (t *Tristate) Set(value bool) {
	if value {
		*t = 1
	} else {
		*t = -1
	}
}

type EnvValues struct {
	LdapURI        string `envconfig:"LDAPURI"`
	LdapBindDn     string `envconfig:"LDAPBINDDN"`
	LdapPassword   string `envconfig:"LDAPPASSWORD"`
	LdapTLSReqcert string `envconfig:"LDAPTLS_REQCERT"`
	RawDry         string `envconfig:"DRY"` // Tri state: "" = undefined, else boolean.
	Dry            bool   `envconfig:"DRY"`
	RawColor       string `envconfig:"COLOR"`
	Color          bool   `envconfig:"COLOR"`
	ConfigFile     string `envconfig:"LDAP2PG_CONFIG"`
}

type CommandAction int

const (
	RunAction CommandAction = iota
	ShowHelpAction
	ShowVersionAction
)

type FlagValues struct {
	Color       bool
	NoColor     bool
	MaybeColor  Tristate
	Verbose     int
	Quiet       int
	Dry         bool
	Real        bool
	MaybeDry    Tristate
	ShowHelp    bool
	ShowVersion bool
	ConfigFile  string
}

func loadFlags() FlagValues {
	values := FlagValues{}
	flag.StringVarP(&values.ConfigFile, "config", "c", "", "Path to YAML configuration file.")
	flag.BoolVar(&values.Color, "color", false, "Force color output.")
	flag.BoolVar(&values.NoColor, "no-color", false, "Force plain text output.")
	flag.BoolVarP(&values.ShowHelp, "help", "?", false, "Show this help message and exit.")
	flag.BoolVarP(&values.ShowVersion, "version", "V", false, "Show version and exit.")
	flag.CountVarP(&values.Verbose, "verbose", "v", "Increase log verbosity.")
	flag.CountVarP(&values.Quiet, "quiet", "q", "Increase log verbosity.")
	flag.BoolVarP(&values.Dry, "dry", "n", false, "Don't touch Postgres, just print what to do.")
	flag.BoolVarP(&values.Real, "real", "N", false, "Real mode, apply changes to Postgres instance.")
	flag.Parse()

	// --nocolor prevales.
	if values.Color {
		slog.Debug("Setting color mode.", "source", "flags")
		values.MaybeColor = 1
	}
	if values.NoColor {
		slog.Debug("Setting plain-text mode.", "source", "flags")
		values.MaybeColor = -1
	}

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
	if !config.MaybeDry.Defined() {
		config.MaybeDry.Set(true)
	}
	if !config.MaybeColor.Defined() {
		config.MaybeColor.Set(isatty.IsTerminal(os.Stderr.Fd()))
		slog.Debug("Setting color mode from stderr.", "color", config.Color)
	}
}
