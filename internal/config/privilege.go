package config

import (
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/privileges"
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
	out = privileges.FlattenProfiles(rawMap)
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
