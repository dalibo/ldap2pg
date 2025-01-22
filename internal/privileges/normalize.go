package privileges

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/tree"
	"golang.org/x/exp/maps"
)

func FlattenProfiles(value map[string]interface{}) map[string][]interface{} {
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

// NormalizeGrantRule from loose YAML
//
// Sets default values. Checks some conflicts.
// Hormonize types for DuplicateGrantRules.
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

	err = normalize.Alias(yamlMap, "owners", "owner")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "privileges", "privilege")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "databases", "database")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "schemas", "schema")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "to")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "grantee")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "role")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "objects", "object")
	if err != nil {
		return
	}

	maps.Copy(rule, yamlMap)

	keys := []string{"owners", "privileges", "databases", "schemas", "roles", "objects"}
	for _, k := range keys {
		rule[k], err = normalize.StringList(rule[k])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
	}
	err = normalize.SpuriousKeys(rule, keys...)
	return
}

// DuplicateGrantRules split plurals for mapstructure
func DuplicateGrantRules(yaml map[string]interface{}) (rules []interface{}) {
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
