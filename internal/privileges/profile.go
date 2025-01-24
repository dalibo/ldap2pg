package privileges

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/tree"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slices"
)

// Profile lists privileges to grant.
//
// e.g. readonly Profile lists SELECT on TABLES, USAGE on SCHEMAS, etc.
//
// Rules references profiles by name and generates grant for each privileges in the profile.
type Profile []Privilege

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
	err = checkPrivilegesACL(out)

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

func checkPrivilegesACL(profiles map[string][]interface{}) error {
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
			_, ok = acls[on]
			if !ok {
				return fmt.Errorf("%s[%d]: unknown ACL: %s", name, i, on)
			}
		}
	}
	return nil
}

// Profiles holds privilege groups
type Profiles map[string][]Privilege

// BuildDefaultArg returns the list of (Type, On) couple referenced.
func (rm Profiles) BuildDefaultArg(def string) (out [][]string) {
	for _, refs := range rm {
		for _, ref := range refs {
			if ref.Default != def {
				continue
			}
			out = append(out, []string{ref.On, ref.Type})
		}
	}
	return
}

func (rm Profiles) BuildTypeMaps() (instance, other, defaults TypeMap) {
	all := make(TypeMap)
	other = make(TypeMap)
	defaults = make(TypeMap)
	instance = make(TypeMap)

	for _, privList := range rm {
		for _, priv := range privList {
			var k, t string
			if "" != priv.Default {
				k = strings.ToUpper(priv.Default) + " DEFAULT"
				t = priv.On + "--" + priv.Type
			} else {
				k = priv.On
				t = priv.Type
			}

			all[k] = append(all[k], t)
		}
	}

	for target, types := range all {
		set := mapset.NewSet(types...)
		types := set.ToSlice()
		slices.Sort(types)
		if strings.HasSuffix(target, " DEFAULT") {
			defaults[target] = types
		} else if acls[target].IsGlobal() {
			instance[target] = types
		} else {
			other[target] = types
		}
		slog.Debug("Managing privileges.", "types", types, "on", target)
	}

	return
}
