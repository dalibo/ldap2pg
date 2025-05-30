package config_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/config"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNormalizeWantRule(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	description: Desc
	role: alice
	`)
	var raw any
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeWantRule(raw)
	r.Nil(err)

	_, exists := value["role"]
	r.False(exists, "role key must be renamed to roles")

	untypedRoles, exists := value["roles"]
	r.True(exists, "role key must be renamed to roles")

	roles := untypedRoles.([]any)
	r.Len(roles, 1)
}

func TestNormalizeRules(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	- description: Desc0
	  role: alice
	- description: Desc1
	  roles:
	  - bob
	`)
	var raw any
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeRules(raw)
	r.Nil(err)
	r.Len(value, 2)
}

func TestNormalizeConfig(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	rules:
	- description: Desc0
	  role: alice
	- description: Desc1
	  roles:
	  - bob
	`)
	var raw any
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	config, err := config.NormalizeConfigRoot(raw)
	r.Nil(err)
	syncMap := config["rules"].([]any)
	r.Len(syncMap, 2)
}
