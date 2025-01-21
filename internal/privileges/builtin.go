package privileges

import (
	_ "embed"
)

var (
	//go:embed sql/database.sql
	inspectDatabase string
	//go:embed sql/global-default.sql
	inspectGlobalDefault string
	//go:embed sql/schema-default.sql
	inspectSchemaDefault string
	//go:embed sql/language.sql
	inspectLanguage string
	//go:embed sql/schema.sql
	inspectSchema string
	//go:embed sql/all-functions.sql
	inspectAllFunctions string
	//go:embed sql/all-sequences.sql
	inspectAllSequences string
	//go:embed sql/all-tables.sql
	inspectAllTables string
)

func init() {
	ACLs = make(map[string]privilege)

	Register("instance", "DATABASE", inspectDatabase)
	Register("instance", "LANGUAGE", inspectLanguage)

	Register("database", "SCHEMA", inspectSchema)
	Register(
		"database", "GLOBAL DEFAULT", inspectGlobalDefault,
		`ALTER DEFAULT PRIVILEGES FOR ROLE %%s GRANT %s ON %s TO %%s;`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE %%s REVOKE %s ON %s FROM %%s;`,
	)
	Register(
		"schema", "SCHEMA DEFAULT", inspectSchemaDefault,
		`ALTER DEFAULT PRIVILEGES FOR ROLE %%s IN SCHEMA %%s GRANT %s ON %s TO %%s;`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE %%s IN SCHEMA %%s REVOKE %s ON %s FROM %%s;`,
	)
	Register("schema", "ALL FUNCTIONS IN SCHEMA", inspectAllFunctions)
	Register("schema", "ALL SEQUENCES IN SCHEMA", inspectAllSequences)
	Register("schema", "ALL TABLES IN SCHEMA", inspectAllTables)
}
