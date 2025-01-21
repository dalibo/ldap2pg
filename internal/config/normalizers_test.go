package config_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNormalizeBooleans(t *testing.T) {
	r := require.New(t)

	r.Equal("true", config.NormalizeBoolean("yes"))
	r.Equal("false", config.NormalizeBoolean("OFF"))
	// Noop for non boolean.
	r.Equal(1, config.NormalizeBoolean(1))
	// Noop for effective boolean.
	r.Equal(true, config.NormalizeBoolean(true))
}

func TestNormalizeList(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	values := config.NormalizeList(value)
	r.Equal(1, len(values))

	values = config.NormalizeList([]string{"string", "list"})
	r.Equal(2, len(values))
}

func TestNormalizeStringList(t *testing.T) {
	r := require.New(t)

	value := interface{}("alice")
	values, err := config.NormalizeStringList(value)
	r.Nil(err)
	r.Equal(1, len(values))
	r.Equal("alice", values[0])
}

func TestNormalizeString(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	fallback_owner: owner
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := config.CheckIsString(mapValue["fallback_owner"])
	r.Nil(err)
}

func TestNormalizeWantRule(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	description: Desc
	role: alice
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeWantRule(raw)
	r.Nil(err)

	_, exists := value["role"]
	r.False(exists, "role key must be renamed to roles")

	untypedRoles, exists := value["roles"]
	r.True(exists, "role key must be renamed to roles")

	roles := untypedRoles.([]interface{})
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
	var raw interface{}
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
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	config, err := config.NormalizeConfigRoot(raw)
	r.Nil(err)
	syncMap := config["rules"].([]interface{})
	r.Len(syncMap, 2)
}
