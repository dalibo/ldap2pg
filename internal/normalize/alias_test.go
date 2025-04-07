package normalize_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/stretchr/testify/require"
)

func TestAlias(t *testing.T) {
	r := require.New(t)

	m := map[string]any{
		"role": "alice",
	}
	err := normalize.Alias(m, "roles", "role")
	r.Nil(err)
	_, found := m["role"]
	r.False(found)
	_, found = m["roles"]
	r.True(found)
}

func TestAliasEmpty(t *testing.T) {
	r := require.New(t)

	m := map[string]any{}
	err := normalize.Alias(m, "roles", "role")
	r.Nil(err)
	_, found := m["roles"]
	r.False(found)
}

func TestAliasConflict(t *testing.T) {
	r := require.New(t)

	m := map[string]any{
		"key0":   "alice",
		"alias0": "alice",
	}
	err := normalize.Alias(m, "key0", "alias0")
	r.NotNil(err)
	r.ErrorContains(err, "key0")
	r.ErrorContains(err, "alias0")
}
