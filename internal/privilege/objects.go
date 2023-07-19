package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

// Instance handle privilege on instance-wide objects.
type Instance struct {
	object, inspect string
}

func NewInstance(object, inspect string) Instance {
	return Instance{
		object:  object,
		inspect: inspect,
	}
}

func (p Instance) Databases(_ postgres.DBMap, defaultDatabase string) (out []string) {
	out = append(out, defaultDatabase)
	return
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

func (p Instance) Expand(g Grant, databases postgres.DBMap) (out []Grant) {
	if "__all__" == g.Object {
		for dbname := range databases {
			g := g // copy
			g.Object = dbname
			out = append(out, g)
		}
	} else {
		out = append(out, g)
	}
	return
}

// All holds privileges on all objects in a schema.
type All struct {
	object  string
	inspect string
}

func NewAll(object, inspect string) All {
	return All{
		object:  object,
		inspect: inspect,
	}
}

func (p All) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
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

func (p All) Expand(g Grant, databases postgres.DBMap) (out []Grant) {
	for _, g := range g.ExpandDatabases(maps.Keys(databases)) {
		out = append(out, g.ExpandSchemas(maps.Keys(databases[g.Database].Schemas))...)
	}
	return
}

// Database handles privileges on database-wide objects.
type Database struct {
	object  string
	inspect string
}

func NewDatabase(object, inspect string) Database {
	return Database{
		object:  object,
		inspect: inspect,
	}
}

func (p Database) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
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

func (p Database) Expand(g Grant, databases postgres.DBMap) (out []Grant) {
	for _, g := range g.ExpandDatabases(maps.Keys(databases)) {
		if "__all__" == g.Object {
			for _, s := range databases[g.Database].Schemas {
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
