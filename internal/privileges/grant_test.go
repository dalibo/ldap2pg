package privileges

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/postgres"
	r "github.com/stretchr/testify/require"
)

func TestGrantString(t *testing.T) {
	g := Grant{
		ACL:      "DATABASE",
		Grantee:  "public",
		Type:     "CONNECT",
		Database: "template1",
	}
	r.Equal(t, `CONNECT ON DATABASE template1 TO public`, g.String())

	g = Grant{
		ACL:      "SCHEMA",
		Grantee:  "public",
		Type:     "CREATE",
		Database: "template1",
		Schema:   "public",
	}
	r.Equal(t, `CREATE ON SCHEMA public TO public`, g.String())

	g = Grant{
		ACL:      "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
	}
	r.Equal(t, `SELECT ON TABLE public.table1 TO public`, g.String())

	g = Grant{
		ACL:    "GLOBAL DEFAULT",
		Owner:  "postgres",
		Type:   "SELECT",
		Object: "TABLES",
	}
	r.Equal(t, `GLOBAL DEFAULT FOR postgres SELECT ON TABLES`, g.String())

	g = Grant{
		ACL:    "SCHEMA DEFAULT",
		Owner:  "postgres",
		Type:   "SELECT",
		Object: "TABLES",
		Schema: "public",
	}
	r.Equal(t, `DEFAULT FOR postgres IN SCHEMA public SELECT ON TABLES`, g.String())

	g = Grant{
		ACL:      "TABLE",
		Grantee:  "public",
		Type:     "SELECT",
		Database: "template1",
		Object:   "table1",
		Schema:   "public",
		Partial:  true,
	}
	r.Equal(t, `PARTIAL SELECT ON TABLE public.table1 TO public`, g.String())

	g = Grant{
		ACL:     "LANGUAGE",
		Grantee: "public",
		Type:    "USAGE",
		Object:  "plpgsql",
	}
	r.Equal(t, `USAGE ON LANGUAGE plpgsql TO public`, g.String())

	g = Grant{
		ACL:     "ALL TABLES IN SCHEMA",
		Grantee: "dave",
		Schema:  "public",
		Type:    "",
	}
	r.Equal(t, `ANY ON ALL TABLES IN SCHEMA public TO dave`, g.String())
}

func TestExpandDatabase(t *testing.T) {
	ACL{
		Name:   "DATABASE-WIDE",
		Scope:  "database",
		Grant:  "GRANT <acl> ON <database> TO <grantee>",
		Revoke: "REVOKE <acl> ON <database> FROM <grantee>",
	}.MustRegister()
	defer func() {
		delete(acls, "DATABASE-WIDE")
	}()

	g := Grant{
		ACL:      "DATABASE-WIDE",
		Database: "db0",
	}
	grants := g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)

	grants = g.ExpandDatabase("other-b1")
	r.Len(t, grants, 0)

	postgres.Databases["db0"] = postgres.Database{}
	defer func() {
		delete(postgres.Databases, "db0")
	}()

	g.Database = "__all__"
	grants = g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)
}

func TestExpandDatabaseInstanceWide(t *testing.T) {
	ACL{
		Name:   "INSTANCE-WIDE",
		Scope:  "instance",
		Grant:  "GRANT <acl> ON <database> TO <grantee>",
		Revoke: "REVOKE <acl> ON <database> FROM <grantee>",
	}.MustRegister()
	defer func() {
		delete(acls, "INSTANCE-WIDE")
	}()

	g := Grant{
		ACL:      "INSTANCE-WIDE",
		Database: "db0",
	}
	grants := g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)

	grants = g.ExpandDatabase("other-db1")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)

	postgres.Databases["db0"] = postgres.Database{}
	defer func() {
		delete(postgres.Databases, "db0")
	}()

	g.Database = "__all__"
	grants = g.ExpandDatabase("db0")
	r.Len(t, grants, 1)
	r.Equal(t, "db0", grants[0].Database)
	acls = nil
}

func TestExpandOwners(t *testing.T) {
	g := Grant{
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
	g := Grant{
		Schema: "nsp0",
	}
	grants := g.ExpandSchemas([]string{"nsp0", "nsp1"})
	r.Len(t, grants, 1)
	r.Equal(t, "nsp0", grants[0].Schema)

	g = Grant{
		Schema: "__all__",
	}
	grants = g.ExpandSchemas([]string{"nsp0", "nsp1"})
	r.Len(t, grants, 2)
	r.Equal(t, "nsp0", grants[0].Schema)
	r.Equal(t, "nsp1", grants[1].Schema)
}

func TestFormatQuery(t *testing.T) {
	g := Grant{
		ACL:      "DATABASE",
		Type:     "CONNECT",
		Grantee:  "public",
		Database: "template1",
		Object:   "object",
		Schema:   "nsp",
	}

	q := g.FormatQuery(`GRANT <privilege> ON <acl> <database> TO <grantee>;`)
	r.Equal(t, `GRANT CONNECT ON DATABASE %s TO %s;`, q.Query)
	r.Len(t, q.QueryArgs, 2)

	q = g.FormatQuery(`REVOKE <privilege> ON <acl> <schema>.<object> TO <grantee>;`)
	r.Equal(t, `REVOKE CONNECT ON DATABASE %s.%s TO %s;`, q.Query)
	r.Len(t, q.QueryArgs, 3)
}

func TestFormatDefaultQuery(t *testing.T) {
	g := Grant{
		Owner:    "alice",
		ACL:      "TABLES",
		Type:     "SELECT",
		Grantee:  "public",
		Database: "template1",
		Schema:   "nsp",
	}

	q := g.FormatQuery(`ADP FOR <owner> IN SCHEMA <schema> GRANT <privilege> ON <acl> TO <grantee>;`)
	r.Equal(t, `ADP FOR %s IN SCHEMA %s GRANT SELECT ON TABLES TO %s;`, q.Query)
	r.Len(t, q.QueryArgs, 3)
}
