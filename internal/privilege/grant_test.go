package privilege_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	r "github.com/stretchr/testify/require"
)

func TestGrantString(t *testing.T) {
	g := privilege.Grant{
		Target:   "DATABASE",
		Grantee:  "public",
		Type:     "CONNECT",
		Database: "template1",
	}
	r.Equal(t, `CONNECT ON DATABASE template1 TO public`, g.String())

	g = privilege.Grant{
		Target:   "SCHEMA",
		Grantee:  "public",
		Type:     "CREATE",
		Database: "template1",
		Schema:   "public",
	}
	r.Equal(t, `CREATE ON SCHEMA template1.public TO public`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
	}
	r.Equal(t, `SELECT ON TABLE template1.public.table1 TO public`, g.String())

	g = privilege.Grant{
		Target: "TABLES",
		Owner:  "postgres",
		Type:   "SELECT",
	}
	r.Equal(t, `GLOBAL DEFAULT FOR postgres SELECT ON TABLES`, g.String())

	g = privilege.Grant{
		Target: "TABLES",
		Owner:  "postgres",
		Type:   "SELECT",
		Schema: "public",
	}
	r.Equal(t, `DEFAULT FOR postgres IN SCHEMA public SELECT ON TABLES`, g.String())

	g = privilege.Grant{
		Target:   "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
		Partial:  true,
	}
	r.Equal(t, `PARTIAL SELECT ON TABLE template1.public.table1 TO public`, g.String())

	g = privilege.Grant{
		Target:  "LANGUAGE",
		Grantee: "public",
		Type:    "USAGE",
		Object:  "plpgsql",
	}
	r.Equal(t, `USAGE ON LANGUAGE plpgsql TO public`, g.String())

	g = privilege.Grant{
		Target:  "ALL TABLES IN SCHEMA",
		Grantee: "dave",
		Schema:  "public",
		Type:    "",
	}
	r.Equal(t, `ANY ON ALL TABLES IN SCHEMA public TO dave`, g.String())
}

func TestExpandDatabase(t *testing.T) {
	g := privilege.Grant{
		Database: "db0",
	}
	grants := g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)

	grants = g.ExpandDatabase("db1")
	r.Len(t, grants, 0)

	g = privilege.Grant{
		Database: "__all__",
	}
	grants = g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)
}

func TestExpandOwners(t *testing.T) {
	g := privilege.Grant{
		Database: "db0",
		Schema:   "nsp0",
		Owner:    "__auto__",
		Grantee:  "toto",
	}
	db := postgres.Database{
		Name:  "db0",
		Owner: "o0",
		Schemas: map[string]postgres.Schema{
			"nsp0": {
				Owner: "o1",
				Creators: []string{
					"o0",
					"o1",
					"o2",
				},
			},
		},
	}
	grants := g.ExpandOwners(db)
	r.Len(t, grants, 3)
	r.Equal(t, "o0", grants[0].Owner)
	r.Equal(t, "toto", grants[0].Grantee)
	r.Equal(t, "o1", grants[1].Owner)
	r.Equal(t, "o2", grants[2].Owner)
}

func TestExpandSchema(t *testing.T) {
	g := privilege.Grant{
		Schema: "nsp0",
	}
	grants := g.ExpandSchemas([]string{"nsp0", "nsp1"})
	r.Len(t, grants, 1)
	r.Equal(t, "nsp0", grants[0].Schema)

	g = privilege.Grant{
		Schema: "__all__",
	}
	grants = g.ExpandSchemas([]string{"nsp0", "nsp1"})
	r.Len(t, grants, 2)
	r.Equal(t, "nsp0", grants[0].Schema)
	r.Equal(t, "nsp1", grants[1].Schema)
}
