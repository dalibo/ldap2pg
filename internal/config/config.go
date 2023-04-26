package config

import (
	"os"
	"path"

	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

func FindConfigFile(userValue string) (configpath string) {
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

type Config struct {
	Version   int
	Ldap      LdapConfig
	Postgres  PostgresConfig
	SyncItems []SyncItem `mapstructure:"sync_map"`
}

type LdapConfig struct {
	URI      string
	BindDn   string
	Password string
}

type PostgresConfig struct {
	FallbackOwner       string    `mapstructure:"fallback_owner"`
	DatabasesQuery      RowsOrSQL `mapstructure:"databases_query"`
	ManagedRolesQuery   RowsOrSQL `mapstructure:"managed_roles_query"`
	RolesBlacklistQuery RowsOrSQL `mapstructure:"roles_blacklist_query"`
}

type LdapSearch struct {
	Base   string
	Filter string
}

type RoleRule struct {
	Names    []string
	Options  RoleOptions
	Comments []string
	Parents  []string
}

func New() Config {
	return Config{
		Postgres: PostgresConfig{
			DatabasesQuery: dedent.Dedent(`
			SELECT datname FROM pg_catalog.pg_database
			WHERE datallowconn IS TRUE ORDER BY 1;`),
			RolesBlacklistQuery: []interface{}{"pg_*", "postgres"},
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
		return
	}
	err = c.LoadYaml(root)
	if err != nil {
		return
	}
	return
}

func (c Config) HasLDAPSearches() bool {
	for _, item := range c.SyncItems {
		if "" != item.LdapSearch.Filter {
			return true
		}
	}
	return false
}
