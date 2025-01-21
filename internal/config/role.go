package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
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
		err = normalize.Alias(yamlMap, "names", "name")
		if err != nil {
			return
		}
		err = normalize.Alias(yamlMap, "parents", "parent")
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
		rule["parents"], err = NormalizeMemberships(rule["parents"])
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

	err = CheckSpuriousKeys(&rule, "names", "comment", "parents", "options", "config", "before_create", "after_create")
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
		yamlMap := yaml.(map[string]interface{})
		for k, v := range yamlMap {
			yamlMap[k] = NormalizeBoolean(v)
		}
		maps.Copy(value, yamlMap)
	case nil:
		return
	default:
		return nil, fmt.Errorf("bad type: %T", yaml)
	}

	err = CheckSpuriousKeys(&value, knownKeys...)
	return
}

func NormalizeMemberships(raw interface{}) (memberships []map[string]interface{}, err error) {
	list := NormalizeList(raw)
	memberships = make([]map[string]interface{}, 0, len(list))
	for i, raw := range list {
		membership, err := NormalizeMembership(raw)
		if err != nil {
			return nil, fmt.Errorf("parents[%d]: %w", i, err)
		}
		memberships = append(memberships, membership)
	}
	return
}

func NormalizeMembership(raw interface{}) (value map[string]interface{}, err error) {
	value = make(map[string]interface{})
	// We could add admin, inherit and set to the map

	switch raw := raw.(type) {
	case string:
		value["name"] = raw
	case map[string]interface{}:
		for k, v := range raw {
			value[k] = NormalizeBoolean(v)
		}
	default:
		return nil, fmt.Errorf("bad type: %T", raw)
	}

	if _, ok := value["name"]; !ok {
		return nil, errors.New("missing name")
	}

	err = CheckSpuriousKeys(&value, "name", "inherit", "set", "admin")
	return
}
