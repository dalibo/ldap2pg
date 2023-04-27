package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *ConfigSuite) TestNormalizeList() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	values := config.NormalizeList(value)
	r.Equal(1, len(values))
}

func (suite *ConfigSuite) TestNormalizeStringList() {
	r := suite.Require()

	value := interface{}("alice")
	values, err := config.NormalizeStringList(value)
	r.Nil(err)
	r.Equal(1, len(values))
	r.Equal("alice", values[0])
}

func (suite *ConfigSuite) TestNormalizeAlias() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := config.NormalizeAlias(&mapValue, "roles", "role")
	r.Nil(err)
	_, found := mapValue["role"]
	r.False(found)
	_, found = mapValue["roles"]
	r.True(found)
}

func (suite *ConfigSuite) TestNormalizeAliasEmpty() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	description: No roles
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := config.NormalizeAlias(&mapValue, "roles", "role")
	r.Nil(err)
	_, found := mapValue["roles"]
	r.False(found)
}

func (suite *ConfigSuite) TestNormalizeString() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	fallback_owner: owner
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := config.NormalizeString(mapValue["fallback_owner"])
	r.Nil(err)
}

func (suite *ConfigSuite) TestNormalizeAliasConflict() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	roles: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := config.NormalizeAlias(&mapValue, "roles", "role")
	conflict := err.(*config.KeyConflict)
	r.NotNil(err)
	r.Equal("roles", conflict.Key)
	r.Equal("role", conflict.Conflict)
}

func (suite *ConfigSuite) TestNormalizeRoleRuleString() {
	r := suite.Require()

	value, err := config.NormalizeRoleRule("alice")
	r.Nil(err)

	names, ok := value["names"].([]string)
	r.True(ok)
	r.Equal(1, len(names))
	r.Equal("alice", names[0])
}

func (suite *ConfigSuite) TestNormalizeRoleRuleSingle() {
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

func (suite *ConfigSuite) TestNormalizeRoleComment() {
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

func (suite *ConfigSuite) TestNormalizeRoleOptionsString() {
	r := suite.Require()

	raw := interface{}("SUPERUSER LOGIN")

	value, err := config.NormalizeRoleOptions(raw)
	r.Nil(err)
	r.True(value["SUPERUSER"].(bool))
	r.True(value["LOGIN"].(bool))
}

func (suite *ConfigSuite) TestNormalizeRoleParents() {
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

func (suite *ConfigSuite) TestNormalizeSyncItem() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	description: Desc
	role: alice
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeSyncItem(raw)
	r.Nil(err)

	_, exists := value["role"]
	r.False(exists, "role key must be renamed to roles")

	untypedRoles, exists := value["roles"]
	r.True(exists, "role key must be renamed to roles")

	roles := untypedRoles.([]interface{})
	r.Len(roles, 1)
}

func (suite *ConfigSuite) TestNormalizeSyncMap() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	- description: Desc0
	  role: alice
	- description: Desc1
	  roles:
	  - bob
	`)
	var raw interface{}
	yaml.Unmarshal([]byte(rawYaml), &raw) //nolint:errcheck

	value, err := config.NormalizeSyncMap(raw)
	r.Nil(err)
	r.Len(value, 2)
}

func (suite *ConfigSuite) TestNormalizeConfig() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	sync_map:
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
	syncMap := config["sync_map"].([]interface{})
	r.Len(syncMap, 2)
}
