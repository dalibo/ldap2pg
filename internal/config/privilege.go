package config

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/tree"
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
