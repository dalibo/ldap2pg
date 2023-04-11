package internal_test

import (
	ldap2pg "github.com/dalibo/ldap2pg/internal"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *ConfigSuite) TestLoadYamlNull() {
	r := suite.Require()

	rawYaml := `null`
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	config := ldap2pg.NewConfig()
	err := config.LoadYaml(values)

	r.NotNil(err)
}

func (suite *ConfigSuite) TestLoadDatabasesQuery() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	databases_query: [postgres]
	`)
	var values interface{}
	yaml.Unmarshal([]byte(rawYaml), &values) //nolint:errcheck

	config := ldap2pg.NewConfig()
	r.Equal("databases_query", config.Postgres.DatabasesQuery.Name)

	err := config.LoadYamlPostgres(values)

	r.Nil(err)
	configQuery := config.Postgres.DatabasesQuery.Value.([]interface{})
	r.Equal("postgres", configQuery[0])
}
