package privilege

import (
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Privilege struct {
	Scope   string
	Object  string
	Inspect string
	Grant   string
	Revoke  string
}

func (p Privilege) BuildRevoke(g Grant) (sql string, args []interface{}) {
	sql = fmt.Sprintf(p.Revoke, g.Type)
	if "" != g.Database && "instance" == p.Scope {
		args = append(args, pgx.Identifier{g.Database})
	}
	if "" != g.Object {
		args = append(args, pgx.Identifier{g.Object})
	}
	args = append(args, pgx.Identifier{g.Grantee})
	return sql, args
}

var (
	Map map[string]Privilege
	//go:embed sql/grant-database.sql
	inspectDatabase string
	//go:embed sql/grant-language.sql
	inspectLanguage string
)

func init() {
	Map = make(map[string]Privilege)

	register(
		"instance",
		"DATABASE",
		inspectDatabase,
		`GRANT %s ON DATABASE %%s TO %%s;`,
		`REVOKE %s ON DATABASE %%s FROM %%s;`,
	)
	register(
		"instance",
		"LANGUAGE",
		inspectLanguage,
		`GRANT %s ON LANGUAGE %%s TO %%s;`,
		`REVOKE %s ON LANGUAGE %%s FROM %%s;`,
	)
}

func register(scope, object, inspect, grant, revoke string) {
	Map[object] = Privilege{
		Scope:   scope,
		Object:  object,
		Inspect: inspect,
		Grant:   grant,
		Revoke:  revoke,
	}
}
