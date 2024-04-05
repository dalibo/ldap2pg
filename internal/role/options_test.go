package role_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/role"
	"github.com/stretchr/testify/require"
)

func TestOptionsDiff(t *testing.T) {
	r := require.New(t)

	role.ProcessColumns([]string{
		"rolsuper",
		"rolcreatedb",
		"rolcreaterole",
		"rolinherit",
		"rolreplication",
		"rolconnlimit",
		"rolbypassrls",
		"rolcanlogin",
	}, true)

	o := role.Options{Super: true}
	diff := o.Diff(role.Options{ConnLimit: -1})
	r.Equal("NOSUPERUSER CONNECTION LIMIT -1", diff)
}
