package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/dalibo/ldap2pg/v6/internal/ldap"
	"github.com/dalibo/ldap2pg/v6/internal/postgres"
	"github.com/dalibo/ldap2pg/v6/internal/privileges"
	"github.com/dalibo/ldap2pg/v6/internal/wanted"
	"github.com/jackc/pgx/v5"
)

func FindDotEnvFile(configpath string) string {
	var envpath string
	if configpath == "-" {
		cwd, err := os.Getwd()
		if err != nil {
			slog.Warn("Cannot get current working directory.", "err", err)
			return ""
		}
		envpath = path.Join(cwd, ".env")
	} else {
		envpath = path.Join(path.Dir(configpath), "/.env")
	}
	_, err := os.Stat(envpath)
	if err != nil {
		return ""
	}
	return envpath
}

func FindConfigFile(userValue string) string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"./ldap2pg.yml",
		"./ldap2pg.yaml",
		path.Join(home, "/.config/ldap2pg.yml"),
		path.Join(home, "/.config/ldap2pg.yaml"),
		"/etc/ldap2pg.yml",
		"/etc/ldap2pg.yaml",
		"/etc/ldap2pg/ldap2pg.yml",
		"/etc/ldap2pg/ldap2pg.yaml",
	}
	return FindFile(userValue, candidates)
}

func FindFile(userValue string, candidates []string) (configpath string) {
	if userValue == "-" {
		return "<stdin>"
	}

	if userValue != "" {
		return userValue
	}

	slog.Debug("Searching configuration file in standard locations.")

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
	Ldap       ldap.Config
	Postgres   PostgresConfig
	ACLs       map[string]privileges.ACL `mapstructure:"acls"`
	Privileges map[string]privileges.Profile
	Rules      wanted.Rules `mapstructure:"rules"`
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
		return fmt.Errorf("bad configuration: %w", err)
	}
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		Dump(root)
	}
	err = c.LoadYaml(root)
	if err != nil {
		return
	}

	c.Rules = c.Rules.SplitStaticRules()
	return
}
