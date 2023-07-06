package config

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/tree"
	"golang.org/x/exp/maps"
)

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

// wellknown holds yaml rewrite for wellknown privileges from v5 format to v6.
var wellknown = map[string]interface{}{
	"__connect__": map[string]string{
		"type": "CONNECT",
		"on":   "DATABASE",
	},
	"__temporary__": map[string]string{
		"type": "TEMPORARY",
		"on":   "DATABASE",
	},
	"__create_on_schemas__": map[string]string{
		"type": "CREATE",
		"on":   "SCHEMA",
	},
	"__usage_on_schemas__": map[string]string{
		"type": "USAGE",
		"on":   "SCHEMA",
	},
}

func NormalizePrivilegeRefs(value interface{}) []interface{} {
	list := NormalizeList(value)

	for i, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		ref := wellknown[s]
		if ref == nil {
			continue
		}
		list[i] = ref
	}

	return list
}

func ResolvePrivilegeRefs(value map[string]interface{}) map[string][]interface{} {
	// Map privilege name -> list of privileges to include.
	heritance := make(map[string][]string)
	// Map privilege name -> list of map[type:... on:...] without inclusion.
	refMap := make(map[string][]interface{})

	// Split value map : string items in heritance and maps in refMap.
	for key, item := range value {
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

	// Walk the tree and copy parents refs back to children.
	for _, priv := range tree.Walk(heritance) {
		for _, parent := range heritance[priv] {
			refMap[priv] = append(refMap[priv], refMap[parent]...)
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
