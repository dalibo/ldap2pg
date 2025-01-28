package privileges

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

// instanceACL handle privilege on instanceACL-wide objects.
//
// like databases, roles, parameters, languages, etc.
type instanceACL struct{}

func (a instanceACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	// column order comes from statement:
	// GRANT $type ON $object TO $grantee;
	err = r.Scan(&g.Type, &g.Object, &g.Grantee)
	g.Target = a.object
	return
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

// databaseACL handles privileges on databaseACL-wide objects.
//
// Like schema.
type databaseACL struct{}

func (a databaseACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Object, &g.Grantee)
	g.Target = a.object
	return
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

// schemaAllACL holds privileges on ALL objects in a schema.
//
// Like tables, sequences, etc.
type schemaAllACL struct{}

func (a schemaAllACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Grantee, &g.Partial)
	g.Target = a.object
	return
}

func (schemaAllACL) Expand(g Grant, database postgres.Database) (out []Grant) {
	for _, g := range g.ExpandDatabase(database.Name) {
		out = append(out, g.ExpandSchemas(maps.Keys(database.Schemas))...)
	}
	return
}
