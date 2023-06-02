package privilege

import (
	_ "embed"
)

type Privilege struct {
	Scope   string
	Object  string
	Inspect string
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
	)
	register(
		"instance",
		"LANGUAGE",
		inspectLanguage,
	)
}

func register(scope, object, inspect string) {
	Map[object] = Privilege{
		Scope:   scope,
		Object:  object,
		Inspect: inspect,
	}
}
