package config_test

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/config"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadPrivilege(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	privileges:
	  ro:
	  - type: CONNECT
	    on: DATABASE
	`)
	var value map[string]any
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	c := config.New()
	err := c.LoadYaml(value)
	r.Nil(err)
	r.Len(c.Privileges, 1)
	r.Contains(c.Privileges, "ro")
	p := c.Privileges["ro"]
	r.Len(p, 1)
	r.Equal("CONNECT", p[0].Type)
	r.Equal("DATABASE", p[0].On)
}

func TestStrictYaml(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	postgres:
	  uri: "postgres://user:xx@localhost"
	  roles_blacklist_query: [postgres]
	  databases_query: [nominal]
	ldap:
	  password: "secret"
	`)
	var value map[string]any
	rr := yaml.Unmarshal([]byte(rawYaml), &value)
	slog.Info(fmt.Sprintf("%s", rr))
	slog.Info(fmt.Sprintf("%s", value))

	c := config.New()
	err := c.DecodeYaml(value)
	r.EqualError(err, "decoding failed due to the following error(s):\n\n'Ldap' has invalid keys: password\n'Postgres' has invalid keys: uri")
}
