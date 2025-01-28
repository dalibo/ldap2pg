package privileges

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/normalize"
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
func (a ACL) Register() error {
	var impl acl

	g := Grant{
		Target:  a.Name,
		Type:    "PRIV",
		Grantee: "_grantee_",
	}

	if "GLOBAL DEFAULT" == a.Name {
		g.Owner = "_owner_"
		impl = newGlobalDefault(a.Name)
	} else if "SCHEMA DEFAULT" == a.Name {
		g.Owner = "_owner_"
		impl = newSchemaDefaultACL(a.Name)
	} else if "instance" == a.Scope {
		g.Object = "_object_" // e.g. plpgsql
		impl = newInstanceACL(a.Name)
	} else if "database" == a.Scope {
		g.Object = "_object_" // e.g. pg_catalog
		impl = newDatabaseACL(a.Name)
	} else if a.Scope == "schema" {
		g.Schema = "_schema_"
		impl = newSchemaAllACL(a.Name)
	} else {
		return fmt.Errorf("unknown scope %q", a.Scope)
	}

	impl.Normalize(&g)

	if g.FormatQuery(a.Grant).IsZero() {
		return fmt.Errorf("grant query is invalid")
	}
	if g.FormatQuery(a.Revoke).IsZero() {
		return fmt.Errorf("revoke query is invalid")
	}

	acls[a.Name] = a
	aclImplentations[a.Name] = impl
	return nil
}

// MustRegister ACL
func (a ACL) MustRegister() {
	if err := a.Register(); err != nil {
		panic(fmt.Errorf("ACL: %s: %w", a.Name, err))
	}
}

func NormalizeACLs(yaml interface{}) (interface{}, error) {
	m, ok := yaml.(map[string]interface{})
	if !ok {
		return yaml, fmt.Errorf("must be a map")
	}

	for k, v := range m {
		acl, ok := v.(map[string]interface{})
		if !ok {
			return yaml, fmt.Errorf("%s: must be a map", k)
		}
		err := normalize.SpuriousKeys(acl, "scope", "inspect", "grant", "revoke")
		if err != nil {
			return yaml, fmt.Errorf("%s: %w", k, err)
		}
	}

	return yaml, nil
}

type acl interface {
	inspecter
	normalizer
	Expand(Grant, postgres.Database) []Grant
}

// ACLs registries
var acls = make(map[string]ACL)
var aclImplentations map[string]acl = make(map[string]acl)

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
	for n := range managedACLs {
		acl := acls[n]
		if strings.HasSuffix(n, " DEFAULT") {
			defaultACLs = append(defaultACLs, n)
		} else if acl.Scope == "instance" {
			instancesACLs = append(instancesACLs, n)
		} else {
			databaseACLs = append(databaseACLs, n)
		}
	}
	return
}
