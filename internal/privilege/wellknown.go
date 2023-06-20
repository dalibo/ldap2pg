package privilege

import (
	_ "embed"
)

var (
	Map map[string]Privilege
	//go:embed sql/grant-database.sql
	inspectDatabase string
	//go:embed sql/grant-language.sql
	inspectLanguage string
)

func init() {
	Map = make(map[string]Privilege)

	register("instance", "DATABASE", inspectDatabase)
	register("instance", "LANGUAGE", inspectLanguage)
}

// queries are grant and revoke queries in order.
func register(scope, object, inspect string, queries ...string) {
	var grant, revoke string

	if 0 < len(queries) {
		grant = queries[0]
		queries = queries[1:]
	} else {
		grant = `GRANT %s ON ` + object + ` %%s TO %%s;`
	}

	if 0 < len(queries) {
		revoke = queries[0]
		queries = queries[1:]
	} else {
		revoke = `REVOKE %s ON ` + object + ` %%s FROM %%s;`
	}

	if 0 < len(queries) {
		panic("too many queries")
	}

	Map[object] = Privilege{
		Scope:   scope,
		Object:  object,
		Inspect: inspect,
		Grant:   grant,
		Revoke:  revoke,
	}
}
