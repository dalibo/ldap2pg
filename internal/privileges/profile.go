package privileges

import (
	"fmt"
	"strings"

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
		t := priv.Type
		if acls[priv.On].Uses("owner") {
			// Couple type and object in type. This is hacky.
			// A more elegant way would be to send an array of couple type/object.
			// Not sure if this is worth the effort.
			// See global-default.sql and schema-default.sql for other side.
			t = fmt.Sprintf("%s ON %s", t, priv.Object)
		}
		managedACLs[priv.On] = append(managedACLs[priv.On], t)
	}

	profiles[name] = p
}

func NormalizeProfiles(value interface{}) (map[string][]interface{}, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type")
	}
	for key, value := range m {
		privileges := []interface{}{}
		for _, rawPrivilege := range value.([]interface{}) {
			_, ok := rawPrivilege.(string)
			if ok {
				// profile inclusion
				privileges = append(privileges, rawPrivilege)
				continue
			}
			privilege, err := NormalizePrivilege(rawPrivilege)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			privileges = append(privileges, DuplicatePrivilege(privilege.(map[string]interface{}))...)
		}
		m[key] = privileges
	}
	out := flattenProfiles(m)

	return out, nil
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
