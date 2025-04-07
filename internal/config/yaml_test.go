package config_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/config"
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
