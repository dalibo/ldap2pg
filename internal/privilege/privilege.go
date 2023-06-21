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
	Grant   string
	Revoke  string
}

// Expand handles grants with __all__ databases.
func (p Privilege) Expand(g Grant, databases postgres.DBMap) (grants []Grant) {
	g.Normalize()
	switch p.Scope {
	case "instance":
		if "" == g.Object || "__all__" == g.Object {
			// One time, we may improve this to handle all
			// languages, all foreign data wrapper, etc. For now,
			// we consider __all__ only for databases.
			for dbname := range databases {
				expansion := g // Copy
				expansion.Object = dbname
				grants = append(grants, expansion)
			}
		} else {
			grants = append(grants, g)
		}
	case "database":
		var dbGrants []Grant
		if "" == g.Database || "__all__" == g.Database {
			for dbname := range databases {
				expansion := g // Copy
				expansion.Database = dbname
				dbGrants = append(dbGrants, expansion)
			}
		} else {
			dbGrants = append(dbGrants, g)
		}

		for _, g := range dbGrants {
			if "" == g.Object || "__all__" == g.Object {
				for _, schema := range databases[g.Database].Schemas {
					expansion := g // Copy
					expansion.Object = schema.Name
					grants = append(grants, expansion)
				}
			} else {
				grants = append(grants, g)
			}
		}
	default:
		slog.Debug("Expanding privilege.", "scope", p.Scope)
		panic("unhandled privilege scope")
	}
	return
}

func (p Privilege) BuildRevoke(g Grant, defaultDatabase string) (q postgres.SyncQuery) {
	q.Query = fmt.Sprintf(p.Revoke, g.Type)
	// REVOKE ... ON ... {object} FROM {grantee}
	q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object}, pgx.Identifier{g.Grantee})
	if "instance" == p.Scope {
		q.Database = defaultDatabase
	} else {
		q.Database = g.Database
	}
	q.LogArgs = p.BuildLogArgs(g)
	return
}

func (p Privilege) BuildGrants(g Grant, databases postgres.DBMap, defaultDatabase string) (queries []postgres.SyncQuery) {
	sql := fmt.Sprintf(p.Grant, g.Type)
	grantee := pgx.Identifier{g.Grantee}

	switch p.Scope {
	case "instance":
		grants := p.expandDatabases(g, databases)
		for _, g := range grants {
			q := postgres.SyncQuery{
				LogArgs: p.BuildLogArgs(g),
				// GRANT ... ON ... {object} TO {grantee}
				Query:     sql,
				QueryArgs: []interface{}{pgx.Identifier{g.Object}, grantee},
				Database:  defaultDatabase,
			}
			queries = append(queries, q)
		}
	case "database":
		grants := p.expandDatabases(g, databases)
		for _, g := range grants {
			grants := p.expandSchemas(g, databases)
			for _, g := range grants {
				q := postgres.SyncQuery{
					LogArgs: p.BuildLogArgs(g),
					// GRANT ... ON ... {object} TO {grantee}
					Query:     sql,
					QueryArgs: []interface{}{pgx.Identifier{g.Object}, grantee},
					Database:  g.Database,
				}
				queries = append(queries, q)
			}
		}
	default:
		slog.Debug("Generating grant.", "scope", p.Scope)
		panic("unhandled privilege scope")
	}
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

func (p Privilege) expandSchemas(g Grant, databases postgres.DBMap) (out []Grant) {
	var input string
	// Use object field if expanding databases in database scope.
	if "database" == p.Scope {
		input = g.Object
	} else {
		input = g.Schema
	}

	if "" == input || "__all__" == input {
		for _, s := range databases[g.Database].Schemas {
			g := g // copy
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
	args = append(args,
		strings.ToLower(g.Target), g.Object,
		"role", g.Grantee,
	)
	return
}
