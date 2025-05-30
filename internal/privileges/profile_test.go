package privileges_test

import (
	"strings"
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/privileges"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrivilegeAlias(t *testing.T) {
	r := require.New(t)

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	ro:
	- type: SELECT
	  on: ALL TABLES IN SCHEMA
	- type: USAGE
	  on: SCHEMA
	rw:
	- ro
	- type: SELECT
	  on: ALL SEQUENCES IN SCHEMA
	ddl:
	- rw
	- type: CREATE
	  on: SCHEMA
	`))
	var raw any
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.Nil(err, rawYaml)

	value, err := privileges.NormalizeProfiles(raw)
	r.Nil(err)
	r.Len(value, 3)
	r.Len(value["ro"], 2)
	r.Len(value["rw"], 3)
	r.Len(value["ddl"], 4)
}

func TestBuiltinPrivilege(t *testing.T) {
	r := require.New(t)

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	ro:
	- __select_on_tables__
	`))
	var raw any
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.Nil(err, rawYaml)

	value, err := privileges.NormalizeProfiles(raw)
	r.Nil(err)
	r.Len(value, 1)
	r.Contains(value, "ro")
	ro := value["ro"]
	r.Len(ro, 3)
}

func TestPrivilegeTypes(t *testing.T) {
	r := require.New(t)

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	ro:
	- on: ALL TABLES IN SCHEMA
	  type: [INSERT, UPDATE, DELETE]
	- on: SCHEMA
	  type: CREATE
	rw:
	- ro
	- types: SELECT
	  on: ALL TABLES IN SCHEMA
	- type: USAGE
	  on: SCHEMA
	  `))
	var raw any
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.Nil(err, rawYaml)

	value, err := privileges.NormalizeProfiles(raw)
	r.Nil(err)
	r.Len(value, 2)
	r.Contains(value, "ro")
	ro := value["ro"]
	r.Len(ro, 4)
}
