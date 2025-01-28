// Default privileges for object owners.
package privileges

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type globalDefaultACL struct{}

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

type schemaDefaultACL struct{}

func (schemaDefaultACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// ALTER DEFAULT PRIVILEGES FOR $owner GRANT $type ON $object IN $schema TO $grantee;
	err = r.Scan(&g.Owner, &g.Type, &g.Target, &g.Schema, &g.Grantee)
	return
}

func (schemaDefaultACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		for _, g := range g.ExpandSchemas(maps.Keys(database.Schemas)) {
			out = append(out, g.ExpandOwners(database)...)
		}
	}
	return
}
