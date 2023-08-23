package config

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/wanted"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

func FindFile(userValue string) (configpath string) {
	if "-" == userValue {
		return "<stdin>"
	}

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
	Version    int
	Ldap       LdapConfig
	Postgres   PostgresConfig
	Privileges privilege.RefMap
	SyncMap    wanted.Rules `mapstructure:"rules"`
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
				WHERE datallowconn IS TRUE
				ORDER BY 1;`, pgx.RowTo[string]),
			ManagedRolesQuery: NewSQLQuery[string](`
				SELECT 'public'
				UNION
				SELECT role.rolname
				FROM pg_roles AS role
				ORDER BY 1;`, pgx.RowTo[string]),
			RolesBlacklistQuery: NewYAMLQuery[string](
				"pg_*",
				"postgres",
			),
			SchemasQuery: NewSQLQuery[postgres.Schema](`
				SELECT nspname, rolname
				FROM pg_catalog.pg_namespace
				JOIN pg_catalog.pg_roles ON pg_catalog.pg_roles.oid = nspowner
				-- Ensure ldap2pg can use.
				WHERE has_schema_privilege(CURRENT_USER, nspname, 'USAGE')
					AND nspname NOT LIKE 'pg_%'
					AND nspname <> 'information_schema'
				ORDER BY 1;`, postgres.RowToSchema),
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
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		Dump(root)
	}
	err = c.LoadYaml(root)
	if err != nil {
		return
	}

	c.Postgres.PrivilegesMap = c.Privileges
	c.SyncMap = c.SyncMap.SplitStaticRules()
	return
}
