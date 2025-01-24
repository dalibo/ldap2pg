package privileges

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

// instance handle privilege on instance-wide objects.
//
// like databases, roles, parameters, languages, etc.
type instance struct {
	object, inspect, grant, revoke string
}

func newInstance(object, inspect, grant, revoke string) instance {
	return instance{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p instance) IsGlobal() bool {
	return true
}

func (p instance) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// GRANT $type ON $object TO $grantee;
	err = r.Scan(&g.Type, &g.Object, &g.Grantee)
	g.Target = p.object
	return
}

func (p instance) String() string {
	return p.object
}

func (p instance) Inspect() string {
	return p.inspect
}

func (p instance) Expand(g Grant, _ postgres.Database, databases []string) (out []Grant) {
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

func (p instance) Normalize(g *Grant) {
	// Grant rule sets Database instead of Object.
	if "" == g.Object {
		g.Object = g.Database
	}
	g.Database = ""
	g.Schema = ""
}

func (p instance) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (p instance) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// database handles privileges on database-wide objects.
//
// Like schema.
type database struct {
	object, inspect, grant, revoke string
}

func newDatabase(object, inspect, grant, revoke string) database {
	return database{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p database) IsGlobal() bool {
	return false
}

func (p database) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Object, &g.Grantee)
	g.Target = p.object
	return
}

func (p database) String() string {
	return p.object
}

func (p database) Inspect() string {
	return p.inspect
}

func (p database) Normalize(g *Grant) {
	// Grant rule sets Schema instead of Object.
	if "" == g.Object {
		g.Object = g.Schema
	}
	g.Schema = ""
}

func (p database) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
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

func (p database) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ... {object} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

func (p database) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... {object} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	return
}

// all holds privileges on all objects in a schema.
//
// Like tables, sequences, etc.
type all struct {
	object, inspect, grant, revoke string
}

func newAll(object, inspect, grant, revoke string) all {
	return all{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p all) IsGlobal() bool {
	return false
}

func (p all) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Grantee, &g.Partial)
	g.Target = p.object
	return
}

func (p all) String() string {
	return p.object
}

func (p all) Inspect() string {
	return p.inspect
}

func (p all) Normalize(_ *Grant) {
}

func (p all) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandSchemas(maps.Keys(database.Schemas))...)
	}
	return
}

func (p all) Grant(g Grant) (q postgres.SyncQuery) {
	// GRANT {type} ON ALL ...
	q.Query = fmt.Sprintf(p.grant, g.Type)
	// GRANT ... ON ALL ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (p all) Revoke(g Grant) (q postgres.SyncQuery) {
	// REVOKE {type} ON ALL ...
	q.Query = fmt.Sprintf(p.revoke, g.Type)
	// REVOKE ... ON ... IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
