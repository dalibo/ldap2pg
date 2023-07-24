package privilege

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

// Instance handle privilege on instance-wide objects.
//
// like databases, roles, parameters, languages, etc.
type Instance struct {
	object, inspect, grant, revoke string
}

func NewInstance(object, inspect, grant, revoke string) Instance {
	return Instance{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p Instance) IsGlobal() bool {
	return true
}

func (p Instance) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// GRANT $type ON $object TO $grantee;
	err = r.Scan(&g.Type, &g.Object, &g.Grantee)
	g.Target = p.object
	return
}

func (p Instance) String() string {
	return p.object
}

func (p Instance) Inspect() string {
	return p.inspect
}

func (p Instance) Expand(g Grant, _ postgres.Database, databases []string) (out []Grant) {
	if "__all__" == g.Object {
		for _, dbname := range databases {
			g := g // copy
			g.Object = dbname
			out = append(out, g)
		}
	} else {
		out = append(out, g)
	}
	return
}

func (p Instance) Normalize(g *Grant) {
	// Grant rule sets Database instead of Object.
	if "" == g.Object {
		g.Object = g.Database
	}
	g.Database = ""
	g.Schema = ""
}

func (p Instance) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (p Instance) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// Database handles privileges on database-wide objects.
//
// Like schema.
type Database struct {
	object, inspect, grant, revoke string
}

func NewDatabase(object, inspect, grant, revoke string) Database {
	return Database{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p Database) IsGlobal() bool {
	return false
}

func (p Database) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Object, &g.Grantee)
	g.Target = p.object
	return
}

func (p Database) String() string {
	return p.object
}

func (p Database) Inspect() string {
	return p.inspect
}

func (p Database) Normalize(g *Grant) {
	// Grant rule sets Schema instead of Object.
	if "" == g.Object {
		g.Object = g.Schema
	}
	g.Schema = ""
}

func (p Database) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
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

func (p Database) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (p Database) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// All holds privileges on all objects in a schema.
//
// Like tables, sequences, etc.
type All struct {
	object, inspect, grant, revoke string
}

func NewAll(object, inspect, grant, revoke string) All {
	return All{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p All) IsGlobal() bool {
	return false
}

func (p All) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Grantee, &g.Partial)
	g.Target = p.object
	return
}

func (p All) String() string {
	return p.object
}

func (p All) Inspect() string {
	return p.inspect
}

func (p All) Normalize(_ *Grant) {
}

func (p All) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandSchemas(maps.Keys(database.Schemas))...)
	}
	return
}

func (p All) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ALL ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ALL ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (p All) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
