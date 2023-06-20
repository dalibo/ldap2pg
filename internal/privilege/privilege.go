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
func (p Privilege) Expand(g Grant, databases []postgres.Database) (grants []Grant) {
	g.Normalize()
	switch p.Scope {
	case "instance":
		if "" == g.Object || "__all__" == g.Object {
			// One time, we may improve this to handle all
			// languages, all foreign data wrapper, etc. For now,
			// we consider __all__ only for databases.
			for _, database := range databases {
				expansion := g // Copy
				expansion.Object = database.Name
				grants = append(grants, expansion)
			}
		} else {
			grants = append(grants, g)
		}
	case "database":
		var dbGrants []Grant
		dbMap := make(map[string]postgres.Database)
		expandDatabases := "" == g.Database || "__all__" == g.Database
		for _, database := range databases {
			dbMap[database.Name] = database
			if expandDatabases {
				expansion := g // Copy
				expansion.Database = database.Name
				dbGrants = append(dbGrants, expansion)
			}
		}

		if !expandDatabases {
			dbGrants = append(dbGrants, g)
		}

		for _, g := range dbGrants {
			if "" == g.Object || "__all__" == g.Object {
				for _, schema := range dbMap[g.Database].Schemas {
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

func (p Privilege) BuildGrants(g Grant, databases []postgres.Database, defaultDatabase string) (queries []postgres.SyncQuery) {
	dbMap := make(map[string]postgres.Database)
	sql := fmt.Sprintf(p.Grant, g.Type)
	grantee := pgx.Identifier{g.Grantee}

	switch p.Scope {
	case "instance":
		var objects []string
		if "" == g.Object || "__all__" == g.Object {
			// Loop on all databases.
			for _, db := range databases {
				objects = append(objects, db.Name)
			}
		} else {
			objects = append(objects, g.Object)
		}
		for _, object := range objects {
			q := postgres.SyncQuery{
				LogArgs: p.BuildLogArgs(g),
				// GRANT ... ON ... {object} TO {grantee}
				Query:     sql,
				QueryArgs: []interface{}{pgx.Identifier{object}, grantee},
				Database:  defaultDatabase,
			}
			queries = append(queries, q)
		}
	case "database":
		var dbGrants []Grant
		expandDatabases := "" == g.Database || "__all__" == g.Database
		for _, db := range databases {
			dbMap[db.Name] = db
			if expandDatabases {
				expansion := g // copy
				expansion.Database = db.Name
				dbGrants = append(dbGrants, expansion)
			}
		}
		if !expandDatabases {
			dbGrants = append(dbGrants, g)
		}

		for _, g := range dbGrants {
			var objects []string
			if "" == g.Object || "__all__" == g.Object {
				// Loop all schema
				db := dbMap[g.Database]
				for _, s := range db.Schemas {
					objects = append(objects, s.Name)
				}
			} else {
				objects = append(objects, g.Object)
			}

			for _, object := range objects {
				q := postgres.SyncQuery{
					LogArgs: p.BuildLogArgs(g),
					// GRANT ... ON ... {object} TO {grantee}
					Query:     sql,
					QueryArgs: []interface{}{pgx.Identifier{object}, grantee},
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
