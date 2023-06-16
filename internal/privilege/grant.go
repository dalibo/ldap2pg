package privilege

import (
	"strings"

	"github.com/jackc/pgx/v5"
)

// Grant holds privilege informations from Postgres inspection or Grant rule.
type Grant struct {
	Target   string // Name of the target object: DATABASE, TABLES, etc.
	Grantor  string
	Grantee  string // "" means default privilege for Grantor.
	Type     string
	Database string // "" for instance grant.
	Schema   string // "" for database grant.
	Object   string // "" for both schema and database grants.
	Partial  bool   // Used for ALL TABLES permissions.
}

func RowTo(row pgx.CollectableRow) (g Grant, err error) {
	err = row.Scan(&g.Grantor, &g.Grantee, &g.Type, &g.Database, &g.Schema, &g.Object, &g.Partial)
	return
}

func (g Grant) String() string {
	b := strings.Builder{}
	if g.Partial {
		b.WriteString("PARTIAL ")
	}
	if "" == g.Grantee {
		b.WriteString("DEFAULT ")
	}
	b.WriteString(g.Type)
	b.WriteString(" ON ")
	b.WriteString(g.Target)
	b.WriteByte(' ')

	o := strings.Builder{}
	o.WriteString(g.Database)
	if "" != g.Schema {
		if o.Len() > 0 {
			o.WriteByte('.')
		}
		o.WriteString(g.Schema)
	}
	if "" != g.Object {
		if o.Len() > 0 {
			o.WriteByte('.')
		}
		o.WriteString(g.Object)
	}

	b.WriteString(o.String())
	if "" != g.Grantee {
		b.WriteString(" TO ")
		b.WriteString(g.Grantee)
	}
	if "" != g.Grantor {
		b.WriteString(" GRANTED BY ")
		b.WriteString(g.Grantor)
	}

	return b.String()
}
