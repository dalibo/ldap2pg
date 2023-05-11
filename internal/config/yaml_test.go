package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *Suite) TestLoadPrivilege() {
	r := suite.Require()

	rawYaml := dedent.Dedent(`
	privileges:
	  ro:
	  - type: CONNECT
	    on: DATABASE
	`)
	var value map[string]interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	c := config.New()
	err := c.LoadYaml(value)
	r.Nil(err)
	r.Len(c.Privileges, 1)
	r.Contains(c.Privileges, "ro")
	p := c.Privileges["ro"]
	r.Len(p, 1)
	r.Equal("CONNECT", p[0].Type)
	r.Equal("DATABASE", p[0].On)
}
