package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
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
