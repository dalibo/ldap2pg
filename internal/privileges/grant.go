package privileges

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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
	Owner    string // For default privileges. Empty otherwise.
	Grantee  string
	Target   string // Name of the referenced ACL: DATABASE, TABLES, etc.
	Type     string // Privilege type (USAGE, SELECT, etc.)
	Database string // "" for instance grant.
	Schema   string // "" for database grant.
	Object   string // "" for both schema and database grants.
	Partial  bool   // Used for ALL TABLES permissions.
}

func (g Grant) IsDefault() bool {
	return g.Owner != ""
}

func (g Grant) IsWildcard() bool {
	return g.Type != ""
}

type normalizer interface {
	Normalize(g *Grant)
}

// Normalize ensures grant fields are consistent with privilege scope.
//
// This way grants from wanted state and from inspect are comparables.
func (g *Grant) Normalize() {
	g.ACL().Normalize(g)
}

func (g Grant) ACLName() string {
	if !g.IsDefault() {
		return g.Target
	} else if g.Schema == "" {
		return "GLOBAL DEFAULT"
	}
	return "SCHEMA DEFAULT"
}

func (g Grant) ACL() acl {
	return acls[g.ACLName()]
}

func (g Grant) String() string {
	b := strings.Builder{}
	if g.Partial {
		b.WriteString("PARTIAL ")
	}
	if g.IsDefault() {
		if g.Schema == "" {
			b.WriteString("GLOBAL ")
		}
		b.WriteString("DEFAULT FOR ")
		b.WriteString(g.Owner)
		if g.Schema != "" {
			b.WriteString(" IN SCHEMA ")
			b.WriteString(g.Schema)
		}
		b.WriteByte(' ')
	}
	if g.Type == "" {
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
		if g.Schema != "" {
			if o.Len() > 0 {
				o.WriteByte('.')
			}
			o.WriteString(g.Schema)
		}
		if g.Object != "" {
			if o.Len() > 0 {
				o.WriteByte('.')
			}
			o.WriteString(g.Object)
		}
		b.WriteString(o.String())
	}

	if g.Grantee != "" {
		b.WriteString(" TO ")
		b.WriteString(g.Grantee)
	}

	return b.String()
}

func (g Grant) ExpandDatabase(database string) (out []Grant) {
	if database == g.Database {
		out = append(out, g)
		return
	}

	if g.Database != "__all__" {
		return
	}

	g.Database = database
	out = append(out, g)

	return
}

func (g Grant) ExpandOwners(database postgres.Database) (out []Grant) {
	if g.Owner != "__auto__" {
		out = append(out, g)
		return
	}

	if database.Name != g.Database {
		return
	}

	// Yield default privilege for database owner.
	var schemas []postgres.Schema
	if g.Schema == "" {
		schemas = maps.Values(database.Schemas)
	} else {
		schemas = []postgres.Schema{database.Schemas[g.Schema]}
	}

	creators := mapset.NewSet[string]()
	for _, s := range schemas {
		creators.Append(s.Creators...)
	}
	creatorsList := creators.ToSlice()
	slices.Sort(creatorsList)

	for _, role := range creatorsList {
		if role == g.Grantee {
			// Avoid granting on self.
			continue
		}
		g := g // copy
		g.Owner = role
		out = append(out, g)
	}

	return
}

func (g Grant) ExpandSchemas(schemas []string) (out []Grant) {
	if g.Schema != "__all__" {
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

// Expand grants from rules.
//
// e.g.: instantiate a grant on all databases for each database.
// Same for schemas and owners.
func Expand(in []Grant, acl string, database postgres.Database) (out []Grant) {
	e := acls[acl]
	for _, grant := range in {
		if grant.ACLName() != acl {
			continue
		}
		out = append(out, e.Expand(grant, database)...)
	}
	return
}
