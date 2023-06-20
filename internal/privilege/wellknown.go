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
