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
	//go:embed sql/all-sequences.sql
	inspectAllSequences string
	//go:embed sql/all-tables.sql
	inspectAllTables string
)

func init() {
	// ACLs
	g := `GRANT <privilege> ON <acl> <object> TO <grantee>;`
	r := `REVOKE <privilege> ON <acl> <object> FROM <grantee>;`

	ACL{
		Name:    "DATABASE",
		Scope:   "instance",
		Inspect: inspectDatabase,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()

	ACL{
		Name:    "LANGUAGE",
		Scope:   "instance",
		Inspect: inspectLanguage,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()

	ACL{
		Name:    "SCHEMA",
		Scope:   "database",
		Inspect: inspectSchema,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()

	ACL{
		// implementation is chosed by name instead of scope.
		Name:    "GLOBAL DEFAULT",
		Scope:   "database",
		Inspect: inspectGlobalDefault,
		Grant:   `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> GRANT <privilege> ON <acl> TO <grantee>;`,
		Revoke:  `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> REVOKE <privilege> ON <acl> FROM <grantee>;`,
	}.MustRegister()
	ACL{
		// implementation is chosed by name instead of scope.
		Name:    "SCHEMA DEFAULT",
		Scope:   "schema",
		Inspect: inspectSchemaDefault,
		Grant:   `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> IN SCHEMA <schema> GRANT <privilege> ON <acl> TO <grantee>;`,
		Revoke:  `ALTER DEFAULT PRIVILEGES FOR ROLE <owner> IN SCHEMA <schema> REVOKE <privilege> ON <acl> FROM <grantee>;`,
	}.MustRegister()

	g = `GRANT <privilege> ON <acl> <schema> TO <grantee>;`
	r = `REVOKE <privilege> ON <acl> <schema> FROM <grantee>;`

	ACL{
		Name:    "ALL FUNCTIONS IN SCHEMA",
		Scope:   "schema",
		Inspect: inspectAllFunctions,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL SEQUENCES IN SCHEMA",
		Scope:   "schema",
		Inspect: inspectAllSequences,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()
	ACL{
		Name:    "ALL TABLES IN SCHEMA",
		Scope:   "schema",
		Inspect: inspectAllTables,
		Grant:   g,
		Revoke:  r,
	}.MustRegister()

	// profiles
	registerRelationBuiltinProfile("sequences", "select", "update", "usage")
	registerRelationBuiltinProfile("tables", "delete", "insert", "select", "truncate", "update", "references", "trigger")
	registerRelationBuiltinProfile("functions", "execute")
}

// BuiltinsProfiles holds yaml rewrite for BuiltinsProfiles privileges from v5 format to v6.
//
// Exported for doc generation.
var BuiltinsProfiles = map[string]interface{}{
	"__connect__": []interface{}{map[string]interface{}{
		"type": "CONNECT",
		"on":   "DATABASE",
	}},
	"__temporary__": []interface{}{map[string]interface{}{
		"type": "TEMPORARY",
		"on":   "DATABASE",
	}},
	"__create_on_schemas__": []interface{}{map[string]interface{}{
		"type": "CREATE",
		"on":   "SCHEMA",
	}},
	"__usage_on_schemas__": []interface{}{map[string]interface{}{
		"type": "USAGE",
		"on":   "SCHEMA",
	}},
	"__all_on_schemas__": []interface{}{
		"__create_on_schemas__",
		"__usage_on_schemas__",
	},
}

// registerRelationBuiltinProfile generates dunder privileges profiles and privilege groups.
//
// example: __all_on_tables__, __select_on_tables_, etc.
func registerRelationBuiltinProfile(class string, types ...string) {
	CLASS := strings.ToUpper(class)
	all := []interface{}{}
	for _, privType := range types {
		TYPE := strings.ToUpper(privType)
		BuiltinsProfiles["__default_"+privType+"_on_"+class+"__"] = []interface{}{map[string]interface{}{
			"default": "global",
			"type":    TYPE,
			"on":      CLASS,
		}, map[string]interface{}{
			"default": "schema",
			"type":    TYPE,
			"on":      CLASS,
		}}
		BuiltinsProfiles["__"+privType+"_on_all_"+class+"__"] = []interface{}{map[string]interface{}{
			"type": TYPE,
			"on":   "ALL " + CLASS + " IN SCHEMA",
		}}
		BuiltinsProfiles["__"+privType+"_on_"+class+"__"] = []interface{}{
			"__default_" + privType + "_on_" + class + "__",
			"__" + privType + "_on_all_" + class + "__",
		}
		all = append(all, "__"+privType+"_on_"+class+"__")
	}
	BuiltinsProfiles["__all_on_"+class+"__"] = all
}
