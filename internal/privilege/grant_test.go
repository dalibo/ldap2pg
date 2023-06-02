package privilege_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/stretchr/testify/require"
)

func TestGrantString(t *testing.T) {
	g := privilege.Grant{
		Target:   "DATABASE",
		Grantor:  "postgres",
		Grantee:  "public",
		Type:     "CONNECT",
		Database: "template1",
	}
	require.Equal(t, `CONNECT ON DATABASE template1 TO public GRANTED BY postgres`, g.String())

	g = privilege.Grant{
		Target:   "SCHEMA",
		Grantor:  "postgres",
		Grantee:  "public",
		Type:     "CREATE",
		Database: "template1",
		Schema:   "public",
	}
	require.Equal(t, `CREATE ON SCHEMA template1.public TO public GRANTED BY postgres`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantor:  "postgres",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
	}
	require.Equal(t, `SELECT ON TABLE template1.public.table1 TO public GRANTED BY postgres`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantor:  "postgres",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
	}
	require.Equal(t, `DEFAULT SELECT ON TABLE template1.public.table1 GRANTED BY postgres`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantor:  "postgres",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
		Partial:  true,
	}
	require.Equal(t, `PARTIAL SELECT ON TABLE template1.public.table1 TO public GRANTED BY postgres`, g.String())

	g = privilege.Grant{
		Target:  "LANGUAGE",
		Grantor: "postgres",
		Grantee: "public",
		Type:    "USAGE",
		Object:  "plpgsql",
	}
	require.Equal(t, `USAGE ON LANGUAGE plpgsql TO public GRANTED BY postgres`, g.String())
}
