package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *Suite) TestRoleRulesString() {
	r := suite.Require()

	value, err := config.NormalizeRoleRule("alice")
	r.Nil(err)

	names, ok := value["names"].([]string)
	r.True(ok)
	r.Equal(1, len(names))
	r.Equal("alice", names[0])
}

func (suite *Suite) TestRoleRulesSingle() {
	r := suite.Require()

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

func (suite *Suite) TestRolesComment() {
	r := suite.Require()

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

func (suite *Suite) TestRoleOptionsString() {
	r := suite.Require()

	raw := interface{}("SUPERUSER LOGIN")

	value, err := config.NormalizeRoleOptions(raw)
	r.Nil(err)
	r.True(value["SUPERUSER"].(bool))
	r.True(value["LOGIN"].(bool))
}

func (suite *Suite) TestRoleParents() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	name: toto
	parents: groupe
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeRoleRule(raw)
	r.Nil(err)
	parents := value["parents"].([]string)
	r.Equal(1, len(parents))
	r.Equal("groupe", parents[0])
}
