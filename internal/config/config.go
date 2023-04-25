package config

import (
	"os"
	"os/user"
	"path"

	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

type Config struct {
	Version  int
	Ldap     LdapConfig
	Postgres PostgresConfig
	SyncMap  []SyncItem `mapstructure:"sync_map"`
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

type SyncItem struct {
	Description string
	LdapSearch  interface{}
	RoleRules   []RoleRule `mapstructure:"roles"`
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

func (config *Config) Load(path string) (err error) {
	slog.Debug("Loading YAML configuration.")

	yamlData, err := ReadYaml(path)
	if err != nil {
		return
	}
	err = config.checkVersion(yamlData)
	if err != nil {
		return
	}
	root, err := NormalizeConfigRoot(yamlData)
	if err != nil {
		return
	}
	err = config.LoadYaml(root)
	if err != nil {
		return
	}
	return
}

func FindConfigFile(userValue string) (configpath string) {
	if "" != userValue {
		return userValue
	}

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

func (c Config) HasLDAPSearches() bool {
	for _, item := range c.SyncMap {
		if item.LdapSearch != nil {
			return true
		}
	}
	return false
}
