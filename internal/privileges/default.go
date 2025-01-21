// Default privileges for object owners.
package privileges

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type globalDefault struct {
	object, inspect, grant, revoke string
}

func newGlobalDefault(object, inspect, grant, revoke string) globalDefault {
	return globalDefault{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p globalDefault) String() string {
	return p.object
}

func (p globalDefault) IsGlobal() bool {
	return false
}

func (p globalDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $target TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Grantee)
	return
}

func (p globalDefault) Inspect() string {
	return p.inspect
}

func (p globalDefault) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandOwners(database)...)
	}
	return
}

func (p globalDefault) Normalize(_ *Grant) {
}

func (p globalDefault) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

func (p globalDefault) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {target} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Grantee})
	return
}

type schemaDefault struct {
	object, inspect, grant, revoke string
}

func newSchemaDefault(object, inspect, grant, revoke string) schemaDefault {
	return schemaDefault{
		object:  object,
		inspect: inspect,
		grant:   grant,
		revoke:  revoke,
	}
}

func (p schemaDefault) IsGlobal() bool {
	return false
}

func (p schemaDefault) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object IN $schema TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Schema, &g.Grantee)
	return
}

func (p schemaDefault) String() string {
	return p.object
}

func (p schemaDefault) Inspect() string {
	return p.inspect
}

func (p schemaDefault) Expand(g Grant, database postgres.Database, _ []string) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		for _, g := range g.ExpandSchemas(maps.Keys(database.Schemas)) {
			out = append(out, g.ExpandOwners(database)...)
		}
	}
	return
}

func (p schemaDefault) Normalize(_ *Grant) {
}

func (p schemaDefault) Grant(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... GRANT {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.grant, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ... IN SCHEMA {schema} ... TO {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}

func (p schemaDefault) Revoke(g Grant) (q postgres.SyncQuery) {
	// ALTER DEFAULT PRIVILEGES ... REVOKE {type} ON {object} ...
	// Unlike regular privileges, object is a keyword parameterized by grant.
	q.Query = fmt.Sprintf(p.revoke, g.Type, g.Target)
	// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} IN SCHEMA {schema} ... FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner}, pgx.Identifier{g.Schema}, pgx.Identifier{g.Grantee})
	return
}
