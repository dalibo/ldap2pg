package config_test

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *Suite) TestPrivilegeAlias() {
	r := suite.Require()

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	ro:
	- type: SELECT
	  on: ALL TABLES
	- type: USAGE
	  on: SCHEMAS
	rw:
	- ro
	- type: SELECT
	  on: ALL SEQUENCES
	ddl:
	- rw
	- type: CREATE
	  on: SCHEMAS
	`))
	var raw interface{}
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.Nil(err, rawYaml)
	rawMap := raw.(map[string]interface{})

	value := config.ResolvePrivilegeRefs(rawMap)
	r.Len(value, 3)
	r.Len(value["ro"], 2)
	r.Len(value["rw"], 3)
	r.Len(value["ddl"], 4)
}

func (suite *Suite) TestBuiltinPrivilege() {
	r := suite.Require()

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	ro:
	- __select_on_tables__
	`))
	var raw interface{}
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.Nil(err, rawYaml)
	rawMap := raw.(map[string]interface{})

	value := config.ResolvePrivilegeRefs(rawMap)
	r.Len(value, 1)
	r.Contains(value, "ro")
	ro := value["ro"]
	r.Len(ro, 2)
}
