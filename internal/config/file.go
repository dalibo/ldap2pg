package config

import (
	"fmt"
	"os"
	"path"

	"github.com/dalibo/ldap2pg/internal/sync"
	"golang.org/x/exp/slog"
)

func FindFile(userValue string) (configpath string) {
	if "" != userValue {
		return userValue
	}

	slog.Debug("Searching configuration file in standard locations.")
	home, _ := os.UserHomeDir()
	candidates := []string{
		"./ldap2pg.yml",
		"./ldap2pg.yaml",
		path.Join(home, "/.config/ldap2pg.yml"),
		path.Join(home, "/.config/ldap2pg.yaml"),
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

// Config holds the YAML configuration. Not the flags.
type Config struct {
	yaml     map[string]interface{}
	Version  int
	Ldap     LdapConfig
	Postgres PostgresConfig
	SyncMap  sync.Map `mapstructure:"sync_map"`
}

type LdapConfig struct {
	URI      string
	BindDn   string
	Password string
}

// New initiate a config structure with defaults.
func New() Config {
	return Config{
		Postgres: PostgresConfig{
			DatabasesQuery: NewSQLQuery[string](`
				SELECT datname FROM pg_catalog.pg_database
				WHERE datallowconn IS TRUE ORDER BY 1;`),
			RolesBlacklistQuery: NewYAMLQuery[string](
				"pg_*",
				"postgres",
			),
		},
	}
}

func Load(path string) (Config, error) {
	c := New()
	err := c.Load(path)
	return c, err
}

func (c *Config) Load(path string) (err error) {
	slog.Debug("Loading YAML configuration.")

	yamlData, err := ReadYaml(path)
	if err != nil {
		return
	}
	err = c.checkVersion(yamlData)
	if err != nil {
		return
	}
	root, err := NormalizeConfigRoot(yamlData)
	if err != nil {
		return fmt.Errorf("YAML error: %w", err)
	}
	err = c.LoadYaml(root)
	if err != nil {
		return
	}

	c.SyncMap = c.SyncMap.SplitStaticRules()
	c.yaml = root
	return
}
