package ldap2pg_test

import (
	"github.com/dalibo/ldap2pg/internal/ldap2pg"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *ConfigSuite) TestLoadYamlNull() {
	r := suite.Require()

	rawYaml := `null`
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	config := ldap2pg.Config{}
	err := config.LoadYaml(values)

	r.NotNil(err)
}

func (suite *ConfigSuite) TestLoadDatabasesQuery() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	postgres:
	  databases_query: [postgres]
	`)
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	config := ldap2pg.Config{}
	err := config.LoadYaml(values)

	r.Nil(err)
	configQuery := config.Postgres.DataBasesQuery.([]interface{})
	r.Equal("postgres", configQuery[0])
}
