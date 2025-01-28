// Default privileges for object owners.
package privileges

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type globalDefaultACL struct {
	object, grant, revoke string
}

func newGlobalDefault(object, grant, revoke string) globalDefaultACL {
	return globalDefaultACL{
		object: object,
		grant:  grant,
		revoke: revoke,
	}
}

func (a globalDefaultACL) String() string {
	return a.object
}

func (globalDefaultACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $target TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Grantee)
	return
}

func (globalDefaultACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandOwners(database)...)
	}
	return
}

func (globalDefaultACL) Normalize(_ *Grant) {
}

func (a globalDefaultACL) Grant(g Grant) postgres.SyncQuery {
	return g.FormatQuery(a.grant)
}

func (a globalDefaultACL) Revoke(g Grant) postgres.SyncQuery {
	return g.FormatQuery(a.revoke)
}

type schemaDefaultACL struct {
	object, grant, revoke string
}

func newSchemaDefaultACL(object, grant, revoke string) schemaDefaultACL {
	return schemaDefaultACL{
		object: object,
		grant:  grant,
		revoke: revoke,
	}
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

func (a schemaDefaultACL) Grant(g Grant) postgres.SyncQuery {
	return g.FormatQuery(a.grant)
}

func (a schemaDefaultACL) Revoke(g Grant) postgres.SyncQuery {
	return g.FormatQuery(a.revoke)
}
