package privileges

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

// instanceACL handle privilege on instanceACL-wide objects.
//
// like databases, roles, parameters, languages, etc.
type instanceACL struct {
	object, inspect, grant, revoke string
}

func newInstanceACL(object, inspect, grant, revoke string) instanceACL {
	return instanceACL{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (a instanceACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// GRANT $type ON $object TO $grantee;
	err = r.Scan(&g.Type, &g.Object, &g.Grantee)
	g.Target = a.object
	return
}

func (a instanceACL) String() string {
	return a.object
}

func (a instanceACL) Inspect() string {
	return a.inspect
}

func (instanceACL) Expand(g Grant, _ postgres.Database) (out []Grant) {
	if "__all__" == g.Object {
		// Expand __all__ to all databases.
		for dbname := range postgres.Databases {
			g := g // copy
			g.Object = dbname
			out = append(out, g)
		}
	} else {
		out = append(out, g)
	}
	return
}

func (instanceACL) Normalize(g *Grant) {
	// Grant rule sets Database instead of Object.
	if "" == g.Object {
		g.Object = g.Database
	}
	g.Database = ""
	g.Schema = ""
}

func (a instanceACL) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(a.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (a instanceACL) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ...
	q.Query = fmt.Sprintf(a.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// databaseACL handles privileges on databaseACL-wide objects.
//
// Like schema.
type databaseACL struct {
	object, inspect, grant, revoke string
}

func newDatabaseACL(object, inspect, grant, revoke string) databaseACL {
	return databaseACL{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (a databaseACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Object, &g.Grantee)
	g.Target = a.object
	return
}

func (a databaseACL) String() string {
	return a.object
}

func (a databaseACL) Inspect() string {
	return a.inspect
}

func (databaseACL) Normalize(g *Grant) {
	// Grant rule sets Schema instead of Object.
	if "" == g.Object {
		g.Object = g.Schema
	}
	g.Schema = ""
}

func (databaseACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		if "__all__" == g.Object {
			for _, s := range database.Schemas {
				g := g // copy
				g.Object = s.Name
				out = append(out, g)
			}
		} else {
			out = append(out, g)
		}
	}
	return
}

func (a databaseACL) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(a.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (a databaseACL) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(a.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// schemaACL holds privileges on schemaACL objects in a schema.
//
// Like tables, sequences, etc.
type schemaACL struct {
	object, inspect, grant, revoke string
}

func newSchemaACL(object, inspect, grant, revoke string) schemaACL {
	return schemaACL{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (a schemaACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Grantee, &g.Partial)
	g.Target = a.object
	return
}

func (a schemaACL) String() string {
	return a.object
}

func (a schemaACL) Inspect() string {
	return a.inspect
}

func (schemaACL) Normalize(_ *Grant) {
}

func (schemaACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandSchemas(maps.Keys(database.Schemas))...)
	}
	return
}

func (a schemaACL) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ALL ...
	q.Query = fmt.Sprintf(a.grant, g.Type)
	// GRANT ... ON ALL ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (a schemaACL) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(a.revoke, g.Type)
	// REVOKE ... ON ... IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
