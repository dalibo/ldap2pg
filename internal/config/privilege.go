package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"github.com/dalibo/ldap2pg/internal/tree"
	"golang.org/x/exp/maps"
)

func (c *Config) DropPrivileges() {
	slog.Debug("Dropping privilege configuration.")
	maps.Clear(c.Privileges)
	c.Rules = c.Rules.DropGrants()
}

func (c Config) ArePrivilegesManaged() bool {
	return 0 < len(c.Privileges)
}

func NormalizePrivileges(value interface{}) (out map[string][]interface{}, err error) {
	rawMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type")
	}
	for key, value := range rawMap {
		privilegeRefs := []interface{}{}
		for _, rawPrivilegeRef := range value.([]interface{}) {
			privilegeRef, ok := rawPrivilegeRef.(map[string]interface{})
			if !ok {
				// should be a string, referencing another profile for inclusion.
				privilegeRefs = append(privilegeRefs, rawPrivilegeRef)
				continue
			}

			err := normalize.Alias(privilegeRef, "types", "type")
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			privilegeRef["types"] = normalize.List(privilegeRef["types"])
			privilegeRefs = append(privilegeRefs, DuplicatePrivilegeRef(privilegeRef)...)
		}
		rawMap[key] = privilegeRefs
	}
	out = ResolvePrivilegeRefs(rawMap)
	err = CheckPrivilegesACL(out)

	return
}

func DuplicatePrivilegeRef(yaml map[string]interface{}) (privileges []interface{}) {
	for _, singleType := range yaml["types"].([]interface{}) {
		privilege := make(map[string]interface{})
		privilege["type"] = singleType
		for key, value := range yaml {
			if "types" == key {
				continue
			}
			privilege[key] = value
		}
		privileges = append(privileges, privilege)
	}
	return
}

func CheckPrivilegesACL(profiles map[string][]interface{}) error {
	for name, profile := range profiles {
		for i, ref := range profile {
			refMap := ref.(map[string]interface{})
			on, ok := refMap["on"].(string)
			if !ok {
				return fmt.Errorf("%s[%d]: missing ACL", name, i)
			}
			_, ok = refMap["default"]
			if ok {
				continue
			}
			_, ok = privileges.ACLs[on]
			if !ok {
				return fmt.Errorf("%s[%d]: unknown ACL: %s", name, i, on)
			}
		}
	}
	return nil
}

// BuiltinsProfiles holds yaml rewrite for BuiltinsProfiles privileges from v5 format to v6.
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

func init() {
	registerRelationBuiltins("sequences", "select", "update", "usage")
	registerRelationBuiltins("tables", "delete", "insert", "select", "truncate", "update", "references", "trigger")
	registerRelationBuiltins("functions", "execute")
}

// registerRelationBuiltins generates dunder privileges profiles and privilege groups.
//
// example: __all_on_tables__, __select_on_tables_, etc.
func registerRelationBuiltins(class string, types ...string) {
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

func ResolvePrivilegeRefs(value map[string]interface{}) map[string][]interface{} {
	// Map privilege name -> list of privileges to include.
	heritance := make(map[string][]string)
	// Map privilege name -> list of map[type:... on:...] without inclusion.
	refMap := make(map[string][]interface{})

	// copyRefs moves string items in heritance map and ref maps in refMap.
	copyRefs := func(refs map[string]interface{}) {
		for key, item := range refs {
			list := item.([]interface{})
			for _, item := range list {
				s, ok := item.(string)
				if ok {
					heritance[key] = append(heritance[key], s)
				} else {
					refMap[key] = append(refMap[key], item)
				}
			}
		}
	}

	// First copy builtins
	copyRefs(BuiltinsProfiles)
	copyRefs(value)

	// Walk the tree and copy parents refs back to children.
	for _, priv := range tree.Walk(heritance) {
		for _, parent := range heritance[priv] {
			refMap[priv] = append(refMap[priv], refMap[parent]...)
		}
	}

	// Remove builtin
	for key := range refMap {
		if strings.HasPrefix(key, "__") {
			delete(refMap, key)
		}
	}

	return refMap
}
