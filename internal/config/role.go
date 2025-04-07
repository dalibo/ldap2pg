package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
	"golang.org/x/exp/maps"
)

func NormalizeRoleRule(yaml any) (rule map[string]any, err error) {
	rule = map[string]any{
		"comment": "Managed by ldap2pg",
		"options": map[string]any{},
		"parents": []string{},
	}

	switch yaml := yaml.(type) {
	case string:
		rule["names"] = []string{yaml}
	case map[string]any:
		err = normalize.Alias(yaml, "names", "name")
		if err != nil {
			return
		}
		err = normalize.Alias(yaml, "parents", "parent")
		if err != nil {
			return
		}

		maps.Copy(rule, yaml)

		names, ok := rule["names"]
		if ok {
			rule["names"], err = normalize.StringList(names)
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

	err = normalize.SpuriousKeys(rule, "names", "comment", "parents", "options", "config", "before_create", "after_create")
	return
}

// Normalize one rule with a list of names to a list of rules with a single
// name.
func DuplicateRoleRules(yaml map[string]any) (rules []map[string]any) {
	for _, name := range yaml["names"].([]string) {
		rule := make(map[string]any)
		rule["name"] = name
		for key, value := range yaml {
			if key == "names" {
				continue
			}
			rule[key] = value
		}
		rules = append(rules, rule)
	}
	return
}

func NormalizeRoleOptions(yaml any) (value map[string]any, err error) {
	// Normal form of role options is a map with SQL token as key and
	// boolean or int value.
	value = map[string]any{
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

	switch yaml := yaml.(type) {
	case string:
		tokens := strings.Split(yaml, " ")
		for _, token := range tokens {
			if token == "" {
				continue
			}
			value[strings.TrimPrefix(token, "NO")] = !strings.HasPrefix(token, "NO")
		}
	case map[string]any:
		for k, v := range yaml {
			yaml[k] = normalize.Boolean(v)
		}
		maps.Copy(value, yaml)
	case nil:
		return
	default:
		return nil, fmt.Errorf("bad type: %T", yaml)
	}

	err = normalize.SpuriousKeys(value, knownKeys...)
	return
}

func NormalizeMemberships(raw any) (memberships []map[string]any, err error) {
	list := normalize.List(raw)
	memberships = make([]map[string]any, 0, len(list))
	for i, raw := range list {
		membership, err := NormalizeMembership(raw)
		if err != nil {
			return nil, fmt.Errorf("parents[%d]: %w", i, err)
		}
		memberships = append(memberships, membership)
	}
	return
}

func NormalizeMembership(raw any) (value map[string]any, err error) {
	value = make(map[string]any)
	// We could add admin, inherit and set to the map

	switch raw := raw.(type) {
	case string:
		value["name"] = raw
	case map[string]any:
		for k, v := range raw {
			value[k] = normalize.Boolean(v)
		}
	default:
		return nil, fmt.Errorf("bad type: %T", raw)
	}

	if _, ok := value["name"]; !ok {
		return nil, errors.New("missing name")
	}

	err = normalize.SpuriousKeys(value, "name", "inherit", "set", "admin")
	return
}
