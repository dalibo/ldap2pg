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
