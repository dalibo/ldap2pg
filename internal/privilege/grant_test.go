package privilege_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/stretchr/testify/require"
)

func TestGrantString(t *testing.T) {
	g := privilege.Grant{
		Target:   "DATABASE",
		Grantee:  "public",
		Type:     "CONNECT",
		Database: "template1",
	}
	require.Equal(t, `CONNECT ON DATABASE template1 TO public`, g.String())

	g = privilege.Grant{
		Target:   "SCHEMA",
		Grantee:  "public",
		Type:     "CREATE",
		Database: "template1",
		Schema:   "public",
	}
	require.Equal(t, `CREATE ON SCHEMA template1.public TO public`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
	}
	require.Equal(t, `SELECT ON TABLE template1.public.table1 TO public`, g.String())

	g = privilege.Grant{
		Target: "TABLES",
		Owner:  "postgres",
		Type:   "SELECT",
	}
	require.Equal(t, `DEFAULT FOR postgres SELECT ON TABLES`, g.String())

	g = privilege.Grant{
		Target: "TABLES",
		Owner:  "postgres",
		Type:   "SELECT",
		Schema: "public",
	}
	require.Equal(t, `DEFAULT FOR postgres IN SCHEMA public SELECT ON TABLES`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
		Partial:  true,
	}
	require.Equal(t, `PARTIAL SELECT ON TABLE template1.public.table1 TO public`, g.String())

	g = privilege.Grant{
		Target:  "LANGUAGE",
		Grantee: "public",
		Type:    "USAGE",
		Object:  "plpgsql",
	}
	require.Equal(t, `USAGE ON LANGUAGE plpgsql TO public`, g.String())
}
