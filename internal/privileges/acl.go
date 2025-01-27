package privileges

import (
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
)

// ACL holds an ACL definition.
//
// An ACL is defined by a Scope and queries to inspect, grant and revoke items.
type ACL struct {
	Name    string
	Scope   string
	Inspect string
	Grant   string
	Revoke  string
}

// Register ACL
//
// scope is one of instance, database, schema.
// Determines de granularity and relevant fields of the privilege.
//
// Grant and Revoke queries may be generated from Name.
func (a ACL) Register() {
	if a.Grant == "" {
		slog.Debug("Building GRANT query.", "acl", a.Name)
		a.Grant = `GRANT %s ON ` + a.Name + ` %%s TO %%s;`
	}

	if a.Revoke == "" {
		slog.Debug("Building REVOKE query.", "acl", a.Name)
		a.Revoke = `REVOKE %s ON ` + a.Name + ` %%s FROM %%s;`
	}

	var impl acl

	if "GLOBAL DEFAULT" == a.Name {
		impl = newGlobalDefault(a.Name, a.Inspect, a.Grant, a.Revoke)
	} else if "SCHEMA DEFAULT" == a.Name {
		impl = newSchemaDefaultACL(a.Name, a.Inspect, a.Grant, a.Revoke)
	} else if "instance" == a.Scope {
		impl = newInstanceACL(a.Name, a.Inspect, a.Grant, a.Revoke)
	} else if "database" == a.Scope {
		impl = newDatabaseACL(a.Name, a.Inspect, a.Grant, a.Revoke)
	} else if a.Scope == "schema" {
		impl = newSchemaAllACL(a.Name, a.Inspect, a.Grant, a.Revoke)
	} else {
		panic("unsupported acl scope")
	}
	acls[a.Name] = impl
}

func NormalizeACLs(yaml interface{}) (interface{}, error) {
	return yaml, nil
}

type acl interface {
	inspecter
	normalizer
	Expand(Grant, postgres.Database) []Grant
	revoker
	granter
}

// ACLs registry
var acls map[string]acl

// managedACLs registry
//
// Lists all managed ACL and for each ACL, the managed privilege types.
// e.g. TABLES = [SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER]
//
// RegisterProfiles feed this map.
//
// Use this map to determine what to inspect and synchronize.
// Actually, use SplitManagedACLs to synchronize managed ACL by scope.
var managedACLs = map[string][]string{}

// SplitManagedACLs by scope
func SplitManagedACLs() (instancesACLs, databaseACLs, defaultACLs []string) {
	for object := range managedACLs {
		switch acls[object].(type) {
		case instanceACL:
			instancesACLs = append(instancesACLs, object)
		case globalDefaultACL, schemaDefaultACL:
			defaultACLs = append(defaultACLs, object)
		default:
			databaseACLs = append(databaseACLs, object)
		}
	}
	return
}
