package ldap2pg

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Action     CommandAction
	ConfigFile string
	LogLevel   log.Level
	Ldap       struct {
		URI      string
		BindDn   string
		Password string
	}
	Postgres PostgresQueries
}

type PostgresQueries struct {
	DatabasesQuery      Query
	ManagedRolesQuery   Query
	RolesBlacklistQuery Query
}

func NewConfig() Config {
	return Config{
		Action: RunAction,
		// Default to current LogLevel.
		LogLevel: log.GetLevel(),
		Postgres: PostgresQueries{
			DatabasesQuery: Query{
				Name: "databases_query",
			},
			ManagedRolesQuery: Query{
				Name: "managed_roles_query",
			},
			RolesBlacklistQuery: Query{
				Name: "roles_blacklist_query",
				// Inject Static value as returned by YAML
				Default: []interface{}{"pg_*", "postgres"},
			},
		},
	}
}

func (config *Config) Load() (err error) {
	log.Debug("Loading Flag values.")
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

	log.Debug("Loading Environment values.")
	var envValues EnvValues
	envconfig.MustProcess("", &envValues)
	config.LoadEnv(envValues)

	log.Debug("Loading YAML configuration.")
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

	return
}

func (config *Config) FindConfigFile() (configpath string) {
	log.Debug("Searching configuration file in standard locations.")
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
			log.
				WithField("path", candidate).
				Debug("Found configuration file.")
			return candidate
		}
		log.
			WithField("path", candidate).
			WithField("error", err).
			Debug("Ignoring configuration file.")
	}

	return ""
}

var levels []log.Level = []log.Level{
	log.TraceLevel,
	log.DebugLevel,
	log.InfoLevel,
	log.WarnLevel,
	log.ErrorLevel,
	log.FatalLevel,
	log.PanicLevel,
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
		log.
			WithField("source", "flags").
			WithField("level", config.LogLevel.String()).
			Debug("Setting log level.")
	}

	if values.ConfigFile != "" {
		log.
			WithField("source", "flags").
			WithField("path", values.ConfigFile).
			Debug("Setting config file.")
		config.ConfigFile = values.ConfigFile
	}
}

func (config *Config) LoadEnv(values EnvValues) {
	if values.LdapURI != "" {
		log.
			WithField("source", "env").
			WithField("value", values.LdapURI).
			Debug("Setting LDAPURI.")
		config.Ldap.URI = values.LdapURI
	}

	if values.LdapBindDn != "" {
		log.
			WithField("value", values.LdapBindDn).
			WithField("source", "env").
			Debug("Setting LDAPBINDDN.")
		config.Ldap.BindDn = values.LdapBindDn
	}

	if values.LdapPassword != "" {
		log.
			WithField("source", "env").
			Debug("Setting LDAPPASSWORD.")
		config.Ldap.Password = values.LdapPassword
	}

	if config.ConfigFile == "" && values.ConfigFile != "" {
		log.
			WithField("source", "env").
			WithField("path", values.ConfigFile).
			Debug("Setting config file.")
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

func (config *Config) LoadDefaults() {
	config.Postgres.DatabasesQuery.SetDefault()
	config.Postgres.RolesBlacklistQuery.SetDefault()
}
