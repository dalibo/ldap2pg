package normalize_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIsString(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	fallback_owner: owner
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	mapValue := value.(map[string]interface{})
	err := normalize.IsString(mapValue["fallback_owner"])
	r.Nil(err)
}
