package privilege

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
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
		if "schema" == p.Scope {
			// [GRANT|REVOKE] ... IN SCHEMA {schema} ...
			q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Schema})
		} else {
			// [GRANT|REVOKE] ... ON ... {object} ...
			q.QueryArgs = append(q.QueryArgs, pgx.Identifier{g.Object})
		}
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

func (p Privilege) BuildLogArgs(g Grant) (args []interface{}) {
	if g.IsDefault() {
		args = append(args,
			"owner", g.Owner,
			"class", g.Target,
		)
	} else {
		if "" == g.Object {
			args = append(args, "object", g.Target)
		} else {
			args = append(args, strings.ToLower(g.Target), g.Object)
		}
		if "schema" == p.Scope {
			args = append(args, "schema", g.Schema)
		}
	}
	args = append(args, "role", g.Grantee)
	if "instance" != p.Scope {
		args = append(args, "database", g.Database)
	}
	return
}
