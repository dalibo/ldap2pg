package role

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsDiff(t *testing.T) {
	r := require.New(t)

	ProcessColumns([]string{
		"rolsuper",
		"rolcreatedb",
		"rolcreaterole",
		"rolinherit",
		"rolreplication",
		"rolconnlimit",
		"rolbypassrls",
		"rolcanlogin",
	}, true)

	o := Options{Super: true}
	diff := o.Diff(Options{ConnLimit: -1})
	r.Equal("NOSUPERUSER CONNECTION LIMIT -1", diff)
}

func TestUnhandledOptions(t *testing.T) {
	r := require.New(t)

	ProcessColumns([]string{
		"rolsuper",
		"rolcreatedb",
		"rolcreaterole",
		"rolinherit",
		"rolreplication",
		"rolconnlimit",
		"rolbypassrls",
		"rolcanlogin",
		"rolcustomopt",
	}, false)

	r.NotContains(instanceColumns.order, "rolcustomopt")
	o := Options{Super: true}
	diff := o.Diff(Options{ConnLimit: -1})
	r.Equal("CONNECTION LIMIT -1", diff)
}
