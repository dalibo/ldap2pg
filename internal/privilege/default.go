// Default privileges for object owners.
package privilege

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type GlobalDefault struct {
	object, inspect, grant, revoke string
}

func NewGlobalDefault(object, inspect, grant, revoke string) GlobalDefault {
	return GlobalDefault{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p GlobalDefault) String() string {
	return p.object
}

func (p GlobalDefault) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
}

func (p GlobalDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $target TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Grantee)
	return
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

func (p GlobalDefault) Normalize(_ *Grant) {
}

func (p GlobalDefault) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

func (p GlobalDefault) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

type SchemaDefault struct {
	object, inspect, grant, revoke string
}

func NewSchemaDefault(object, inspect, grant, revoke string) SchemaDefault {
	return SchemaDefault{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p SchemaDefault) Databases(m postgres.DBMap, _ string) []string {
	return maps.Keys(m)
}

func (p SchemaDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object IN $schema TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Schema, &g.Grantee)
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

func (p SchemaDefault) Normalize(_ *Grant) {
}

func (p SchemaDefault) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... GRANT {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (p SchemaDefault) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... REVOKE {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
