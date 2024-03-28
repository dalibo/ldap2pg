package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/tree"
	"golang.org/x/exp/maps"
)

func (c *Config) DropPrivileges() {
	slog.Debug("Dropping privilege configuration.")
	maps.Clear(c.Privileges)
	c.SyncMap = c.SyncMap.DropGrants()
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
		rawMap[key] = NormalizePrivilegeRefs(value)
	}

	out = ResolvePrivilegeRefs(rawMap)

	return
}

// BuiltinsProfiles holds yaml rewrite for BuiltinsProfiles privileges from v5 format to v6.
var BuiltinsProfiles = map[string]interface{}{
	"__connect__": []interface{}{map[string]string{
		"type": "CONNECT",
		"on":   "DATABASE",
	}},
	"__temporary__": []interface{}{map[string]string{
		"type": "TEMPORARY",
		"on":   "DATABASE",
	}},
	"__create_on_schemas__": []interface{}{map[string]string{
		"type": "CREATE",
		"on":   "SCHEMA",
	}},
	"__usage_on_schemas__": []interface{}{map[string]string{
		"type": "USAGE",
		"on":   "SCHEMA",
	}},
	"__all_on_schemas__": []interface{}{
		"__create_on_schemas__",
		"__usage_on_schema__",
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
		BuiltinsProfiles["__default_"+privType+"_on_"+class+"__"] = []interface{}{map[string]string{
			"default": "global",
			"type":    TYPE,
			"on":      CLASS,
		}, map[string]string{
			"default": "schema",
			"type":    TYPE,
			"on":      CLASS,
		}}
		BuiltinsProfiles["__"+privType+"_on_all_"+class+"__"] = []interface{}{map[string]string{
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

func NormalizePrivilegeRefs(value interface{}) []interface{} {
	list := NormalizeList(value)

	for i, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		ref := BuiltinsProfiles[s]
		if ref == nil {
			continue
		}
		refMap, ok := ref.(map[string]string)
		if !ok {
			continue
		}
		list[i] = refMap
	}

	return list
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

func NormalizeGrantRule(yaml interface{}) (rule map[string]interface{}, err error) {
	rule = map[string]interface{}{
		"owners":    "__auto__",
		"schemas":   "__all__",
		"databases": "__all__",
	}

	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type")
	}

	err = NormalizeAlias(&yamlMap, "owners", "owner")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "privileges", "privilege")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "databases", "database")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "schemas", "schema")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "roles", "to")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "roles", "grantee")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "roles", "role")
	if err != nil {
		return
	}
	err = NormalizeAlias(&yamlMap, "objects", "object")
	if err != nil {
		return
	}

	maps.Copy(rule, yamlMap)

	keys := []string{"owners", "privileges", "databases", "schemas", "roles", "objects"}
	for _, k := range keys {
		rule[k], err = NormalizeStringList(rule[k])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
	}
	err = CheckSpuriousKeys(&rule, keys...)
	return
}

func DuplicateGrantRules(yaml map[string]interface{}) (rules []map[string]interface{}) {
	keys := []string{"owners", "databases", "schemas", "roles", "objects", "privileges"}
	keys = lists.Filter(keys, func(s string) bool {
		return len(yaml[s].([]string)) > 0
	})
	fields := [][]string{}
	for _, k := range keys {
		fields = append(fields, yaml[k].([]string))
	}
	for combination := range lists.Product(fields...) {
		rule := map[string]interface{}{}
		for i, k := range keys {
			rule[strings.TrimSuffix(k, "s")] = combination[i]
		}
		rules = append(rules, rule)
	}
	return
}
