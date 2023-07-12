package privilege

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Privilege holds queries and metadata to manage a set of privilege type.
//
// Not to be confused with Grant. Privilege is an abstract representation of
// ACL on a kind of object. There is no object in PostgreSQL that represent
// this concept.
type Privilege struct {
	Scope   string
	Object  string
	Inspect string
	// Grant and revoke queries are double format string. The first
	// formatting is for object (TABLE, SCHEMA, etc.) and type (SELECT,
	// INSERT, etc.). The second formating is for grant parameters, usually
	// SQL identifiers: schema, object name, grantee, etc..
	Grant  string
	Revoke string
}

func (p Privilege) IsZero() bool {
	return "" == p.Inspect
}

func (p Privilege) IsDefault() bool {
	return strings.HasSuffix(p.Object, "DEFAULT")
}

// Expand handles grants with __all__ databases.
func (p Privilege) Expand(g Grant, databases postgres.DBMap) (grants []Grant) {
	switch p.Scope {
	case "instance":
		grants = p.expandDatabases(g, databases)
	case "database":
		dbGrants := p.expandDatabases(g, databases)
		for _, g := range dbGrants {
			schemaGrants := p.expandSchemas(g, databases)
			for _, g := range schemaGrants {
				grants = append(grants, p.expandOwners(g, databases)...)
			}
		}
	default:
		slog.Debug("Expanding privilege.", "scope", p.Scope)
		panic("unhandled privilege scope")
	}
	return
}

func (p Privilege) BuildRevoke(g Grant, defaultDatabase string) postgres.SyncQuery {
	return p.BuildQuery(g, p.Revoke, defaultDatabase)
}

func (p Privilege) BuildGrant(g Grant, defaultDatabase string) postgres.SyncQuery {
	return p.BuildQuery(g, p.Grant, defaultDatabase)
}

func (p Privilege) BuildQuery(g Grant, format, defaultDatabase string) (q postgres.SyncQuery) {
	if p.IsDefault() {
		// ALTER DEFAULT PRIVILEGES ... [GRANT|REVOKE] {type} ON {object} ...
		// Unlike regular privileges, object is a keyword parameterized by grant.
		q.Query = fmt.Sprintf(format, g.Type, g.Target)
		// ALTER DEFAULT PRIVILEGES FOR ROLE {owner} ...
		q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Owner})
		if "" != g.Schema {
			// ALTER DEFAULT PRIVILEGES FOR {owner} IN SCHEMA {schema} ...
			q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema})
		}
	} else {
		// [GRANT|REVOKE] {type} ON ...
		q.Query = fmt.Sprintf(format, g.Type)
		// [GRANT|REVOKE] ... ON ... {object} ...
		q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object})
	}

	// ... [FROM|TO] {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Grantee})
	if "instance" == p.Scope {
		q.Database = defaultDatabase
	} else {
		q.Database = g.Database
	}
	q.LogArgs = p.BuildLogArgs(g)
	return
}

func (p Privilege) expandDatabases(g Grant, databases postgres.DBMap) (out []Grant) {
	var input string
	// Use object field if expanding databases in instance scope.
	if "instance" == p.Scope {
		input = g.Object
	} else {
		input = g.Database
	}

	if "" == input || "__all__" == input {
		for dbname := range databases {
			g := g // copy
			if "instance" == p.Scope {
				g.Object = dbname
			} else {
				g.Database = dbname
			}
			out = append(out, g)
		}
	} else {
		out = append(out, g)
	}
	return
}

func (p Privilege) expandOwners(g Grant, databases postgres.DBMap) (out []Grant) {
	if !p.IsDefault() {
		g.Owner = ""
		out = append(out, g)
		return
	}

	if "__auto__" != g.Owner {
		out = append(out, g)
		return
	}

	database := databases[g.Database]
	g.Owner = database.Owner
	out = append(out, g)

	if "" == g.Schema {
		for _, s := range database.Schemas {
			g := g // copy
			g.Owner = s.Owner
			out = append(out, g)
		}
	} else {
		g.Owner = database.Schemas[g.Schema].Owner
	}

	if "" == g.Owner {
		slog.Debug("Expand owners.", "grant", g, "database", database)
		panic("no owner")
	}

	return
}

func (p Privilege) expandSchemas(g Grant, databases postgres.DBMap) (out []Grant) {
	var input string
	// Use object field if expanding databases in database scope.
	if "database" == p.Scope && !p.IsDefault() {
		input = g.Object
	} else {
		input = g.Schema
	}

	if "__all__" == input {
		for _, s := range databases[g.Database].Schemas {
			g := g // copy
			// Should never happen for default privilege. See Normalize.
			if "database" == p.Scope {
				g.Object = s.Name
			} else {
				g.Schema = s.Name
			}
			out = append(out, g)
		}
	} else {
		out = append(out, g)
	}
	return
}

func (p Privilege) BuildLogArgs(g Grant) (args []interface{}) {
	args = append(args, "type", g.Type)
	if "instance" != p.Scope {
		args = append(args, "database", g.Database)
	}
	if g.IsDefault() {
		args = append(args,
			"owner", g.Owner,
			"class", g.Target,
		)
	} else {
		args = append(args, strings.ToLower(g.Target), g.Object)
	}
	args = append(args, "role", g.Grantee)
	return
}
