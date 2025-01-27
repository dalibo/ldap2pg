package privileges

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/tree"
)

// Profile lists privileges to grant.
//
// e.g. readonly Profile lists SELECT on TABLES, USAGE on SCHEMAS, etc.
//
// Rules references profiles by name and generates grant for each privileges in the profile.
type Profile []Privilege

func (p Profile) Register(name string) {
	for _, priv := range p {
		on := priv.On
		t := priv.Type
		if priv.IsDefault() {
			on = fmt.Sprintf("%s DEFAULT", strings.ToUpper(priv.Default))
			t = fmt.Sprintf("%s--%s", priv.On, strings.ToUpper(t))
		}
		managedACLs[on] = append(managedACLs[on], t)
	}

	profiles[name] = p
}

func NormalizeProfiles(value interface{}) (out map[string][]interface{}, err error) {
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
			privilegeRefs = append(privilegeRefs, DuplicatePrivilege(privilegeRef)...)
		}
		rawMap[key] = privilegeRefs
	}
	out = flattenProfiles(rawMap)

	return
}

func flattenProfiles(value map[string]interface{}) map[string][]interface{} {
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

var profiles = make(map[string]Profile)
