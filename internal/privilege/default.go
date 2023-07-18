// Default privileges for object owners.
package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type GlobalDefault struct {
	object  string
	inspect string
}

func NewGlobalDefault(object, inspect string) GlobalDefault {
	return GlobalDefault{
		object:  object,
		inspect: inspect,
	}
}

func (p GlobalDefault) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
}

func (p GlobalDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Object, &g.Grantee)
	// Instead of p.object, get the object class from the inspect query.
	g.Target = g.Object
	return
}

func (p GlobalDefault) String() string {
	return p.object
}

func (p GlobalDefault) Inspect() string {
	return p.inspect
}

func (p GlobalDefault) Expand(g Grant, databases postgres.DBMap) (out []Grant) {
	for _, g := range g.ExpandDatabases(maps.Keys(databases)) {
		out = append(out, g.ExpandOwners(databases)...)
	}
	return
}

type SchemaDefault struct {
	object  string
	inspect string
}

func NewSchemaDefault(object, inspect string) SchemaDefault {
	return SchemaDefault{
		object:  object,
		inspect: inspect,
	}
}

func (p SchemaDefault) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
}

func (p SchemaDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object IN $schema TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Object, &g.Schema, &g.Grantee)
	// Instead of p.object, get the object class from the inspect query.
	g.Target = p.object
	return
}

func (p SchemaDefault) String() string {
	return p.object
}

func (p SchemaDefault) Inspect() string {
	return p.inspect
}

func (p SchemaDefault) Expand(g Grant, databases postgres.DBMap) (out []Grant) {
	for _, g := range g.ExpandDatabases(maps.Keys(databases)) {
		for _, g := range g.ExpandSchemas(maps.Keys(databases[g.Database].Schemas)) {
			out = append(out, g.ExpandOwners(databases)...)
		}
	}
	return
}
