// Default privileges for object owners.
package privileges

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type globalDefaultACL struct {
	object, inspect, grant, revoke string
}

func newGlobalDefault(object, inspect, grant, revoke string) globalDefaultACL {
	return globalDefaultACL{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (a globalDefaultACL) String() string {
	return a.object
}

func (globalDefaultACL) IsGlobal() bool {
	return false
}

func (globalDefaultACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $target TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Grantee)
	return
}

func (a globalDefaultACL) Inspect() string {
	return a.inspect
}

func (globalDefaultACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandOwners(database)...)
	}
	return
}

func (globalDefaultACL) Normalize(_ *Grant) {
}

func (a globalDefaultACL) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(a.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

func (a globalDefaultACL) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(a.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

type schemaDefaultACL struct {
	object, inspect, grant, revoke string
}

func newSchemaDefaultACL(object, inspect, grant, revoke string) schemaDefaultACL {
	return schemaDefaultACL{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (schemaDefaultACL) IsGlobal() bool {
	return false
}

func (schemaDefaultACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object IN $schema TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Schema, &g.Grantee)
	return
}

func (a schemaDefaultACL) String() string {
	return a.object
}

func (a schemaDefaultACL) Inspect() string {
	return a.inspect
}

func (schemaDefaultACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		for _, g := range g.ExpandSchemas(maps.Keys(database.Schemas)) {
			out = append(out, g.ExpandOwners(database)...)
		}
	}
	return
}

func (schemaDefaultACL) Normalize(_ *Grant) {
}

func (a schemaDefaultACL) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... GRANT {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(a.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (a schemaDefaultACL) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... REVOKE {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(a.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
