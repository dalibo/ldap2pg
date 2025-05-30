package normalize_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dalibo/ldap2pg/v6/internal/normalize"
)

func TestBooleans(t *testing.T) {
	r := require.New(t)

	r.Equal("true", normalize.Boolean("yes"))
	r.Equal("false", normalize.Boolean("OFF"))
	// Noop for non boolean.
	r.Equal(1, normalize.Boolean(1))
	// Noop for effective boolean.
	r.Equal(true, normalize.Boolean(true))
}
