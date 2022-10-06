package ldap2pg_test

import (
	"github.com/dalibo/ldap2pg/internal/ldap2pg"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *TestSuite) TestNormalizeList() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	values := ldap2pg.NormalizeList(value)
	r.Equal(1, len(values))
}

func (suite *TestSuite) TestNormalizeStringList() {
	r := suite.Require()

	value := interface{}("alice")
	values, err := ldap2pg.NormalizeStringList(value)
	r.Nil(err)
	r.Equal(1, len(values))
	r.Equal("alice", values[0])
}

func (suite *TestSuite) TestNormalizeAlias() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := ldap2pg.NormalizeAlias(&mapValue, "roles", "role")
	r.Nil(err)
	_, found := mapValue["role"]
	r.False(found)
	_, found = mapValue["roles"]
	r.True(found)
}

func (suite *TestSuite) TestNormalizeAliasEmpty() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	description: No roles
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := ldap2pg.NormalizeAlias(&mapValue, "roles", "role")
	r.Nil(err)
	_, found := mapValue["roles"]
	r.False(found)
}

func (suite *TestSuite) TestNormalizeAliasConflict() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	role: alice
	roles: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := ldap2pg.NormalizeAlias(&mapValue, "roles", "role")
	conflict := err.(*ldap2pg.KeyConflict)
	r.NotNil(err)
	r.Equal("roles", conflict.Key)
	r.Equal("role", conflict.Conflict)
}
