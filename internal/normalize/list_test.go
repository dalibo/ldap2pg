package normalize_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestList(t *testing.T) {
	r := require.New(t)

	rawYaml := dedent.Dedent(`
	role: alice
	`)
	var value interface{}
	yaml.Unmarshal([]byte(rawYaml), &value) //nolint:errcheck

	values := normalize.List(value)
	r.Equal(1, len(values))

	values = normalize.List([]string{"string", "list"})
	r.Equal(2, len(values))
}

func TestStringList(t *testing.T) {
	r := require.New(t)

	value := interface{}("alice")
	values, err := normalize.StringList(value)
	r.Nil(err)
	r.Equal(1, len(values))
	r.Equal("alice", values[0])
}
