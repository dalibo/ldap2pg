package privileges

import (
	_ "embed"
	"strings"
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
	//go:embed sql/all-routines.sql
	inspectAllRoutines string
	//go:embed sql/all-sequences.sql
	inspectAllSequences string
	//go:embed sql/all-tables.sql
	inspectAllTables string
)

func init() {
	// ACLs

	ACL{
		Name:    "DATABASE",
		Scope:   "instance",
		Inspect: inspectDatabase,
		Grant:   `GRANT <privilege> ON <acl> <database> TO <grantee>;`,
		Revoke:  `REVOKE <privilege> ON <acl> <database> FROM <grantee>;`,
	}.MustRegister()

	ACL{
		Name:    "LANGUAGE",
		Scope:   "instance",
		Inspect: inspectLanguage,
		Grant:   `GRANT <privilege> ON <acl> <object> TO <grantee>;`,
		Revoke:  `REVOKE <privilege> ON <acl> <object> FROM <grantee>;`,
	}.MustRegister()

	g := `GRANT <privilege> ON <acl> <schema> TO <grantee>;`
	r := `REVOKE <privilege> ON <acl> <schema> FROM <grantee>;`

	ACL{
		Name:    "SCHEMA",
		Scope:   "database",
		Inspect: inspectSchema,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL FUNCTIONS IN SCHEMA",
		Scope:   "database",
		Inspect: inspectAllFunctions,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL ROUTINES IN SCHEMA",
		Scope:   "database",
		Inspect: inspectAllRoutines,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL SEQUENCES IN SCHEMA",
		Scope:   "database",
		Inspect: inspectAllSequences,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL TABLES IN SCHEMA",
		Scope:   "database",
		Inspect: inspectAllTables,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()

	ACL{
		// implementation is chosed by name instead of scope.
		Name:    "GLOBAL DEFAULT",
		Scope:   "database",
		Inspect: inspectGlobalDefault,
		Grant:   `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> GRANT <privilege> ON <object> TO <grantee>;`,
		Revoke:  `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> REVOKE <privilege> ON <object> FROM <grantee>;`,
	}.MustRegister()
	ACL{
		// implementation is chosed by name instead of scope.
		Name:    "SCHEMA DEFAULT",
		Scope:   "schema",
		Inspect: inspectSchemaDefault,
		Grant:   `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> IN SCHEMA <schema> GRANT <privilege> ON <object> TO <grantee>;`,
		Revoke:  `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> IN SCHEMA <schema> REVOKE <privilege> ON <object> FROM <grantee>;`,
	}.MustRegister()

	// profiles
	registerRelationBuiltinProfile("sequences", "select", "update", "usage")
	registerRelationBuiltinProfile("tables", "delete", "insert", "select", "truncate", "update", "references", "trigger")
	registerRelationBuiltinProfile("routines", "execute")
}

// BuiltinsProfiles holds yaml rewrite for BuiltinsProfiles privileges from v5 format to v6.
//
// Exported for doc generation.
var BuiltinsProfiles = map[string]any{
	"__connect__": []any{map[string]any{
		"type": "CONNECT",
		"on":   "DATABASE",
	}},
	"__temporary__": []any{map[string]any{
		"type": "TEMPORARY",
		"on":   "DATABASE",
	}},
	"__create_on_schemas__": []any{map[string]any{
		"type": "CREATE",
		"on":   "SCHEMA",
	}},
	"__usage_on_schemas__": []any{map[string]any{
		"type": "USAGE",
		"on":   "SCHEMA",
	}},
	"__all_on_schemas__": []any{
		"__create_on_schemas__",
		"__usage_on_schemas__",
	},
	// Privileges on functions has change in 6.5.0.
	// Default on functions is now void.
	// Manage only a EXECUTE ON ALL FUNCTIONS.
	"__execute_on_all_functions__": []any{map[string]any{
		"type": "EXECUTE",
		"on":   "ALL FUNCTIONS IN SCHEMA",
	}},
	"__execute_on_functions__": []any{
		"__execute_on_all_functions__",
	},
	// For backward compatibility.
	"__default_execute_on_functions__": []any{},
	"__all_on_functions__": []any{ // pretty useless.
		"__execute_on_functions__",
	},
}

// registerRelationBuiltinProfile generates dunder privileges profiles and privilege groups.
//
// example: __all_on_tables__, __select_on_tables_, etc.
func registerRelationBuiltinProfile(class string, types ...string) {
	CLASS := strings.ToUpper(class)
	all := []any{}
	for _, privType := range types {
		TYPE := strings.ToUpper(privType)
		BuiltinsProfiles["__default_"+privType+"_on_"+class+"__"] = []any{map[string]any{
			"type":   TYPE,
			"on":     "GLOBAL DEFAULT",
			"object": CLASS,
		}, map[string]any{
			"type":   TYPE,
			"on":     "SCHEMA DEFAULT",
			"object": CLASS,
		}}
		BuiltinsProfiles["__"+privType+"_on_all_"+class+"__"] = []any{map[string]any{
			"type": TYPE,
			"on":   "ALL " + CLASS + " IN SCHEMA",
		}}
		BuiltinsProfiles["__"+privType+"_on_"+class+"__"] = []any{
			"__default_" + privType + "_on_" + class + "__",
			"__" + privType + "_on_all_" + class + "__",
		}
		all = append(all, "__"+privType+"_on_"+class+"__")
	}
	BuiltinsProfiles["__all_on_"+class+"__"] = all
}
