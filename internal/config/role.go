package config

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
)

func NormalizeRoleRule(yaml interface{}) (rule map[string]interface{}, err error) {
	rule = map[string]interface{}{
		"comment": "Managed by ldap2pg",
		"options": map[string]interface{}{},
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
			return nil, errors.New("missing name")
		}
		rule["parents"], err = NormalizeStringList(rule["parents"])
		if err != nil {
			return
		}
		rule["options"], err = NormalizeRoleOptions(rule["options"])
		if err != nil {
			return nil, fmt.Errorf("options: %w", err)
		}
	default:
		return nil, fmt.Errorf("bad type: %T", yaml)
	}

	err = CheckSpuriousKeys(&rule, "names", "comment", "parents", "options", "config")
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
	knownKeys := maps.Keys(value)

	switch yaml.(type) {
	case string:
		s := yaml.(string)
		tokens := strings.Split(s, " ")
		for _, token := range tokens {
			if "" == token {
				continue
			}
			value[strings.TrimPrefix(token, "NO")] = !strings.HasPrefix(token, "NO")
		}
	case map[string]interface{}:
		maps.Copy(value, yaml.(map[string]interface{}))
	case nil:
		return
	default:
		return nil, fmt.Errorf("bad type: %T", yaml)
	}

	err = CheckSpuriousKeys(&value, knownKeys...)
	return
}
