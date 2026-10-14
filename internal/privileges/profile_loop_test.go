package privileges_test

import (
	"strings"
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/privileges"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrivilegeSelfLoop(t *testing.T) {
	r := require.New(t)

	rawYaml := strings.TrimSpace(dedent.Dedent(`
	rw:
	- rw
	- __all_on_tables__
	- __all_on_sequences__
	`))
	var raw any
	err := yaml.Unmarshal([]byte(rawYaml), &raw)
	r.NoError(err)

	value, err := privileges.NormalizeProfiles(raw)
	r.Nil(value)
	r.Error(err)
	r.ErrorContains(err, "privilege profile")
	r.ErrorContains(err, "loop involving rw")
}
