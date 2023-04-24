package config

import (
	"os"
	"os/user"
	"path"

	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

type Config struct {
	Version int
	Ldap    struct {
		URI      string
		BindDn   string
		Password string
	}
	Postgres PostgresConfig
	SyncMap  []SyncItem
}

type PostgresConfig struct {
	FallbackOwner       string
	DatabasesQuery      InspectQuery
	ManagedRolesQuery   InspectQuery
	RolesBlacklistQuery InspectQuery
}

func New() Config {
	return Config{
		Postgres: PostgresConfig{
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

func Load(path string) (Config, error) {
	c := New()
	err := c.Load(path)
	return c, err
}

func (config *Config) Load(path string) (err error) {
	slog.Debug("Loading YAML configuration.")

	yamlValues, err := ReadYaml(path)
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

type EnvValues struct {
	LdapURI        string
	LdapBindDn     string
	LdapPassword   string
	LdapTLSReqcert string
}

func (config *Config) LoadEnv(values EnvValues) {
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
}

func (config *Config) LoadDefaults() {
	config.Postgres.DatabasesQuery.SetDefault()
	config.Postgres.RolesBlacklistQuery.SetDefault()
}
