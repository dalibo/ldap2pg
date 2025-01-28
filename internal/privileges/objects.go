package privileges

import (
	"github.com/jackc/pgx/v5"
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

// databaseACL handles privileges on databaseACL-wide objects.
//
// Like schema.
type databaseACL struct{}

func (a databaseACL) RowTo(r pgx.CollectableRow) (g Grant, err error) {
	err = r.Scan(&g.Type, &g.Schema, &g.Object, &g.Grantee)
	g.Target = a.object
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
