package config_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRoleRulesString(t *testing.T) {
	r := require.New(t)

	value, err := config.NormalizeRoleRule("alice")
	r.Nil(err)

	names, ok := value["names"].([]string)
	r.True(ok)
	r.Equal(1, len(names))
	r.Equal("alice", names[0])
}

func TestRoleRulesSingle(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	name: alice
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeRoleRule(raw)
	r.Nil(err)

	rawNames, ok := value["names"]
	r.True(ok)
	names := rawNames.([]string)
	r.Equal(1, len(names))
	r.Equal("alice", names[0])
	r.Equal("Managed by ldap2pg", value["comment"])
}

func TestRolesComment(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	name: alice
	comment: au pays des merveilles.
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeRoleRule(raw)
	r.Nil(err)
	r.Equal([]string{"alice"}, value["names"])
	r.Equal("au pays des merveilles.", value["comment"])
}

func TestRoleOptionsString(t *testing.T) {
	r := require.New(t)

	raw := interface{}("SUPERUSER LOGIN")

	value, err := config.NormalizeRoleOptions(raw)
	r.Nil(err)
	r.True(value["SUPERUSER"].(bool))
	r.True(value["LOGIN"].(bool))
}

func TestRoleParents(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	name: toto
	parents: groupe
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeRoleRule(raw)
	r.Nil(err)
	r.Len(value["parents"], 1)
}

func TestMembership(t *testing.T) {
	r := require.New(t)

	membership, err := config.NormalizeMembership("owners")
	r.Nil(err)
	r.Equal("owners", membership["name"])

	rawYaml := dedent.Dedent(`
	name: owners
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	membership, err = config.NormalizeMembership(raw)
	r.Nil(err)
	r.Equal("owners", membership["name"])
}
