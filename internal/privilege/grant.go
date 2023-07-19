package privilege

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
)

// Grant holds privilege informations from Postgres inspection or Grant rule.
//
// Not to confuse with Privilege. A Grant references an object, a role and a
// privilege via the Target field. It's somewhat like aclitem object in
// PostgreSQL.
//
// When Owner is non-zero, the grant represent a default privilege grant. The
// meansing of Object field change to hold the privilege class : TABLES,
// SEQUENCES, etc. instead of the name of an object.
type Grant struct {
	Target   string // Name of the referenced privilege object: DATABASE, TABLES, etc.
	Owner    string // For default privilege. Empty otherwise.
	Grantee  string
	Type     string // Privilege type (USAGE, SELECT, etc.)
	Database string // "" for instance grant.
	Schema   string // "" for database grant.
	Object   string // "" for both schema and database grants.
	Partial  bool   // Used for ALL TABLES permissions.
}

func (g Grant) IsDefault() bool {
	return "" != g.Owner
}

type Normalizer interface {
	Normalize(g *Grant)
}

// Normalize ensures grant fields are consistent with privilege scope.
//
// This way grants from wanted state and from inspect are comparables.
func (g *Grant) Normalize() {
	g.Privilege().Normalize(g)
}

func (g Grant) Privilege() (p Privilege) {
	if !g.IsDefault() {
		p = Builtins[g.Target]
	} else if "" == g.Schema {
		p = Builtins["GLOBAL DEFAULT"]
	} else {
		p = Builtins["SCHEMA DEFAULT"]
	}
	return
}

func (g Grant) String() string {
	b := strings.Builder{}
	if g.Partial {
		b.WriteString("PARTIAL ")
	}
	if g.IsDefault() {
		b.WriteString("DEFAULT FOR ")
		b.WriteString(g.Owner)
		if "" != g.Schema {
			b.WriteString(" IN SCHEMA ")
			b.WriteString(g.Schema)
		}
		b.WriteByte(' ')
	}
	if "" == g.Type {
		b.WriteString("ANY")
	} else {
		b.WriteString(g.Type)
	}
	b.WriteString(" ON ")
	b.WriteString(g.Target)
	if !g.IsDefault() {
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
	}

	if "" != g.Grantee {
		b.WriteString(" TO ")
		b.WriteString(g.Grantee)
	}

	return b.String()
}

func (g Grant) ExpandDatabases(databases []string) (out []Grant) {
	if "__all__" != g.Database {
		out = append(out, g)
		return
	}

	for _, name := range databases {
		g := g // copy
		g.Database = name
		out = append(out, g)
	}

	return
}

func (g Grant) ExpandOwners(databases postgres.DBMap) (out []Grant) {
	if "__auto__" != g.Owner {
		out = append(out, g)
		return
	}

	// Yield default privilege for database owner.
	database := databases[g.Database]
	g.Owner = database.Owner
	out = append(out, g)

	if "" == g.Schema {
		return
	}

	// Yield default privilege for schema owner.
	g.Owner = database.Schemas[g.Schema].Owner
	out = append(out, g)

	return
}

func (g Grant) ExpandSchemas(schemas []string) (out []Grant) {
	if "__all__" != g.Schema {
		out = append(out, g)
		return
	}

	for _, name := range schemas {
		g := g // copy
		g.Schema = name
		out = append(out, g)
	}

	return
}
