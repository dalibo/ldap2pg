package privileges

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
)

type acl interface {
	inspecter
	normalizer
	Expand(Grant, postgres.Database, []string) []Grant
	revoker
	granter
}

// ACLs registry
var acls map[string]acl

// registerACL an ACL
//
// scope is one of instance, database, namespace.
// Determines de granularity and relevant fields of the privilege.
//
// grant and revoke queries may be generated from object.
func registerACL(scope, object, inspect string, queries ...string) {
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

	var p acl

	if "GLOBAL DEFAULT" == object {
		p = newGlobalDefault(object, inspect, grant, revoke)
	} else if "SCHEMA DEFAULT" == object {
		p = newSchemaDefault(object, inspect, grant, revoke)
	} else if strings.HasPrefix(object, "ALL ") {
		p = newAll(object, inspect, grant, revoke)
	} else if "instance" == scope {
		p = newInstance(object, inspect, grant, revoke)
	} else if "database" == scope {
		p = newDatabase(object, inspect, grant, revoke)
	} else {
		panic("unsupported acl scope")
	}
	acls[object] = p
}
