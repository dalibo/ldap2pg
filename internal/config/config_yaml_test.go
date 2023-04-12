package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *ConfigSuite) TestLoadYamlNull() {
	r := suite.Require()

	rawYaml := `null`
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	c := config.New()
	err := c.LoadYaml(values)

	r.NotNil(err)
}

func (suite *ConfigSuite) TestLoadDatabasesQuery() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	databases_query: [postgres]
	`)
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	c := config.New()
	r.Equal("databases_query", c.Postgres.DatabasesQuery.Name)

	err := c.LoadYamlPostgres(values)

	r.Nil(err)
	configQuery := c.Postgres.DatabasesQuery.Value.([]interface{})
	r.Equal("postgres", configQuery[0])
}
