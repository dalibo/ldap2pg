// Functions to normalize YAML input before processing into data structure.
package config

import (
	"errors"
	"fmt"
	"strings"
)

type KeyConflict struct {
	Key      string
	Conflict string
}

func (err *KeyConflict) Error() string {
	return "YAML alias conflict"
}

type ParseError struct {
	Message string
	Value   interface{}
}

func (err *ParseError) Error() string {
	return err.Message
}

func NormalizeAlias(yaml *map[string]interface{}, key, alias string) (err error) {
	value, hasAlias := (*yaml)[alias]
	if !hasAlias {
		return
	}

	_, hasKey := (*yaml)[key]
	if hasKey {
		err = &KeyConflict{
			Key:      key,
			Conflict: alias,
		}
		return
	}

	delete(*yaml, alias)
	(*yaml)[key] = value
	return
}

func NormalizeList(yaml interface{}) (list []interface{}) {
	list, ok := yaml.([]interface{})
	if !ok {
		list = append(list, yaml)
	}
	return
}

func NormalizeString(yaml interface{}) error {
	_, ok := yaml.(string)
	if !ok && yaml != nil {
		return fmt.Errorf("bad value %v, must be string", yaml)
	}
	return nil
}

func NormalizeStringList(yaml interface{}) (list []string, err error) {
	switch yaml.(type) {
	case nil:
		return
	case string:
		list = append(list, yaml.(string))
	case []interface{}:
		for _, iItem := range yaml.([]interface{}) {
			item, ok := iItem.(string)
			if !ok {
				err = errors.New("Must be string")
			}
			list = append(list, item)
		}
	}
	return
}

func NormalizeRoleRules(yaml interface{}) (rule map[string]interface{}, err error) {
	var names []string
	switch yaml.(type) {
	case string:
		rule = make(map[string]interface{})
		names = append(names, yaml.(string))
		rule["names"] = names
	case map[string]interface{}:
		rule = yaml.(map[string]interface{})
		err = NormalizeAlias(&rule, "names", "name")
		if err != nil {
			return
		}
		names, ok := rule["names"]
		if ok {
			rule["names"], err = NormalizeStringList(names)
			if err != nil {
				return
			}
		} else {
			err = errors.New("Missing name in role rule")
			return
		}
		err = NormalizeAlias(&rule, "parents", "parent")
		if err != nil {
			return
		}
		parents, ok := rule["parents"]
		if ok {
			rule["parents"], err = NormalizeStringList(parents)
			if err != nil {
				return
			}
		} else {
			rule["parents"] = []string{}
		}

		_, ok = rule["comment"]
		if !ok {
			rule["comment"] = "Managed by ldap2pg"
		}

		options := rule["options"]
		rule["options"], err = NormalizeRoleOptions(options)
		if err != nil {
			return
		}
	default:
		err = &ParseError{
			Message: "Invalid role rule YAML",
			Value:   yaml,
		}
	}
	return
}

// Normalize one rule with a list of names to a list of rules with a single
// name.
func DuplicateRoleRules(yaml map[string]interface{}) (rules []map[string]interface{}) {
	for _, name := range yaml["names"].([]string) {
		rule := make(map[string]interface{})
		rule["name"] = name
		for key, value := range yaml {
			if "names" == key {
				continue
			}
			rule[key] = value
		}
		rules = append(rules, rule)
	}
	return
}

func NormalizeRoleOptions(yaml interface{}) (value map[string]interface{}, err error) {
	// Normal form of role options is a map with SQL token as key and
	// boolean or int value.
	value = make(map[string]interface{})

	switch yaml.(type) {
	case string:
		s := yaml.(string)
		tokens := strings.Split(s, " ")
		for _, token := range tokens {
			value[strings.TrimPrefix(token, "NO")] = !strings.HasPrefix(token, "NO")
		}
	case nil:
		return
	default:
		err = &ParseError{
			Message: "invalid role options YAML",
			Value:   yaml,
		}
	}
	return
}

func NormalizeSyncItem(yaml interface{}) (item map[string]interface{}, err error) {
	item, ok := yaml.(map[string]interface{})
	if !ok {
		err = errors.New("invalid sync item type")
		return
	}

	descYaml, ok := item["description"]
	if ok {
		_, ok := descYaml.(string)
		if !ok {
			err = errors.New("sync item description must be string")
			return
		}
	}
	err = NormalizeAlias(&item, "roles", "role")
	if err != nil {
		return
	}
	rawList, exists := item["roles"]
	if exists {
		list := NormalizeList(rawList)
		rules := []interface{}{}
		for _, rawRule := range list {
			var rule map[string]interface{}
			rule, err = NormalizeRoleRules(rawRule)
			if err != nil {
				return
			}
			for _, rule := range DuplicateRoleRules(rule) {
				rules = append(rules, rule)
			}
		}
		item["roles"] = rules
	}

	err = NormalizeAlias(&item, "ldapsearch", "ldap")
	if err != nil {
		return
	}
	iLdapSearch, exists := item["ldapsearch"]
	if exists {
		ldapSearch, ok := iLdapSearch.(map[string]interface{})
		if !ok {
			err = errors.New("invalid ldapsearch type")
			return
		}
		_, ok = ldapSearch["scope"]
		if !ok {
			ldapSearch["scope"] = "sub"
		}
		item["ldapsearch"] = ldapSearch
		joins, ok := ldapSearch["joins"].(map[string]interface{})
		if !ok {
			return
		}
		for attr := range joins {
			joinMap := joins[attr].(map[string]interface{})
			_, ok = joinMap["filter"]
			if !ok {
				joinMap["filter"] = "(objectClass=*)"
			}
			_, ok = joinMap["scope"]
			if !ok {
				joinMap["scope"] = "sub"
			}
		}
	}
	return
}

func NormalizeSyncMap(yaml interface{}) (syncMap []interface{}, err error) {
	rawItems, ok := yaml.([]interface{})
	if !ok {
		err = errors.New("Bad sync_map format")
	}
	for _, rawItem := range rawItems {
		var item interface{}
		item, err = NormalizeSyncItem(rawItem)
		if err != nil {
			return
		}
		syncMap = append(syncMap, item)
	}
	return
}

func NormalizeConfigRoot(yaml interface{}) (config map[string]interface{}, err error) {
	config, ok := yaml.(map[string]interface{})
	if !ok {
		err = errors.New("Bad configuration format")
		return
	}

	section, ok := config["postgres"]
	if ok {
		err = NormalizePostgres(section)
		if err != nil {
			return
		}
	}

	section, ok = config["sync_map"]
	if !ok {
		err = errors.New("Missing sync_map")
		return
	}
	syncMap, err := NormalizeSyncMap(section)
	if err != nil {
		return
	}
	config["sync_map"] = syncMap
	return
}

func NormalizePostgres(yaml interface{}) error {
	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		return fmt.Errorf("bad postgres section, must be a map")
	}

	return NormalizeString(yamlMap["fallback_owner"])
}
