package config_test

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

func (suite *Suite) TestPrivilegeWellknown() {
	r := suite.Require()

	raw := []interface{}{"__connect__"}
	value := config.NormalizePrivilegeRefs(raw)

	ref := value[0].(map[string]string)
	r.Equal("CONNECT", ref["type"])
	r.Equal("DATABASE", ref["on"])
}

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
