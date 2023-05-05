// Functions to normalize YAML input before processing into data structure.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"golang.org/x/exp/maps"
)

type KeyConflict struct {
	Key      string
	Conflict string
}

func (err *KeyConflict) Error() string {
	return fmt.Sprintf("YAML alias conflict between %s and %s", err.Key, err.Conflict)
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
		var search map[string]interface{}
		search, err = NormalizeLdapSearch(iLdapSearch)
		if err != nil {
			return
		}
		item["ldapsearch"] = search
	}
	return
}

func NormalizeLdapSearch(yaml interface{}) (search map[string]interface{}, err error) {
	search, err = NormalizeCommonLdapSearch(yaml)
	if err != nil {
		return
	}
	err = NormalizeAlias(&search, "subsearches", "joins")
	if err != nil {
		return
	}
	subsearches, ok := search["subsearches"].(map[string]interface{})
	if !ok {
		return
	}
	for attr := range subsearches {
		var subsearch map[string]interface{}
		subsearch, err = NormalizeCommonLdapSearch(subsearches[attr])
		if err != nil {
			return
		}
		subsearches[attr] = subsearch
	}
	return
}

func NormalizeCommonLdapSearch(yaml interface{}) (search map[string]interface{}, err error) {
	search = map[string]interface{}{
		"filter": "(objectClass=*)",
		"scope":  "sub",
	}
	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		err = errors.New("invalid ldapsearch type")
		return
	}
	maps.Copy(search, yamlMap)
	search["filter"] = ldap.CleanFilter(search["filter"].(string))
	return
}

func NormalizeRoleRules(yaml interface{}) (rule map[string]interface{}, err error) {
	rule = map[string]interface{}{
		"comment": "Managed by ldap2pg",
		"options": "",
		"parents": []string{},
	}

	switch yaml.(type) {
	case string:
		rule["names"] = []string{yaml.(string)}
	case map[string]interface{}:
		yamlMap := yaml.(map[string]interface{})
		err = NormalizeAlias(&yamlMap, "names", "name")
		if err != nil {
			return
		}
		err = NormalizeAlias(&yamlMap, "parents", "parent")
		if err != nil {
			return
		}

		maps.Copy(rule, yamlMap)

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
		rule["parents"], err = NormalizeStringList(rule["parents"])
		if err != nil {
			return
		}
		rule["options"], err = NormalizeRoleOptions(rule["options"])
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
	value = map[string]interface{}{
		"SUPERUSER":        false,
		"INHERIT":          true,
		"CREATEROLE":       false,
		"CREATEDB":         false,
		"LOGIN":            false,
		"REPLICATION":      false,
		"BYPASSRLS":        false,
		"CONNECTION LIMIT": -1,
	}

	switch yaml.(type) {
	case string:
		s := yaml.(string)
		tokens := strings.Split(s, " ")
		for _, token := range tokens {
			value[strings.TrimPrefix(token, "NO")] = !strings.HasPrefix(token, "NO")
		}
	case map[string]interface{}:
		maps.Copy(value, yaml.(map[string]interface{}))
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
