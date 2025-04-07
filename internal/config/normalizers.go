// Functions to normalize YAML input before processing into data structure.
package config

import (
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"golang.org/x/exp/maps"
)

func NormalizeConfigRoot(yaml any) (config map[string]any, err error) {
	config, ok := yaml.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("bad type: %T", yaml)
	}

	section, ok := config["postgres"]
	if ok {
		err = NormalizePostgres(section)
		if err != nil {
			return config, fmt.Errorf("postgres: %w", err)
		}
	}

	section, ok = config["acls"]
	if ok {
		acls, err := privileges.NormalizeACLs(section)
		if err != nil {
			return config, fmt.Errorf("acls: %w", err)
		}
		config["acls"] = acls
	}

	section, ok = config["privileges"]
	if ok {
		privileges, err := privileges.NormalizeProfiles(section)
		if err != nil {
			return config, fmt.Errorf("privileges: %w", err)
		}
		config["privileges"] = privileges
	}

	err = normalize.Alias(config, "rules", "sync_map")
	if err != nil {
		return
	}
	section, ok = config["rules"]
	if !ok {
		return config, errors.New("missing rules")
	}
	syncMap, err := NormalizeRules(section)
	if err != nil {
		return config, fmt.Errorf("rules: %w", err)
	}
	config["rules"] = syncMap
	return
}

func NormalizePostgres(yaml any) error {
	_, ok := yaml.(map[string]any)
	if !ok {
		return fmt.Errorf("bad type: %T, must be a map", yaml)
	}
	return nil
}

func NormalizeRules(yaml any) (syncMap []any, err error) {
	rawRules, ok := yaml.([]any)
	if !ok {
		return nil, fmt.Errorf("bad type: %T, must be a list", yaml)
	}
	for i, rawRule := range rawRules {
		var item any
		item, err = NormalizeWantRule(rawRule)
		if err != nil {
			return syncMap, fmt.Errorf("item[%d]: %w", i, err)
		}
		syncMap = append(syncMap, item)
	}
	return
}

func NormalizeWantRule(yaml any) (rule map[string]any, err error) {
	rule = map[string]any{
		"description": "",
		"ldapsearch":  map[string]any{},
		"roles":       []any{},
		"grants":      []any{},
	}

	yamlMap, ok := yaml.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("bad type: %T, must be a map", yaml)
	}

	err = normalize.Alias(yamlMap, "ldapsearch", "ldap")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "role")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "grants", "grant")
	if err != nil {
		return
	}

	maps.Copy(rule, yamlMap)

	search, err := NormalizeLdapSearch(rule["ldapsearch"])
	if err != nil {
		return nil, fmt.Errorf("ldapsearch: %w", err)
	}
	rule["ldapsearch"] = search

	list := normalize.List(rule["roles"])
	rules := []any{}
	for i, rawRule := range list {
		var rule map[string]any
		rule, err = NormalizeRoleRule(rawRule)
		if err != nil {
			return nil, fmt.Errorf("roles[%d]: %w", i, err)
		}
		for _, rule := range DuplicateRoleRules(rule) {
			rules = append(rules, rule)
		}
	}
	rule["roles"] = rules

	list = normalize.List(rule["grants"])
	rules = []any{}
	for i, rawRule := range list {
		var rule map[string]any
		rule, err = privileges.NormalizeGrantRule(rawRule)
		if err != nil {
			return nil, fmt.Errorf("grants[%d]: %w", i, err)
		}
		rules = append(rules, privileges.DuplicateGrantRules(rule)...)
	}
	rule["grants"] = rules

	err = normalize.SpuriousKeys(rule, "description", "ldapsearch", "roles", "grants")
	return
}

func NormalizeLdapSearch(yaml any) (search map[string]any, err error) {
	search, err = NormalizeCommonLdapSearch(yaml)
	if err != nil {
		return
	}
	err = normalize.Alias(search, "subsearches", "joins")
	if err != nil {
		return
	}
	err = normalize.SpuriousKeys(search, "base", "filter", "scope", "subsearches", "on_unexpected_dn")
	if err != nil {
		return
	}

	subsearches, ok := search["subsearches"].(map[string]any)
	if !ok {
		return
	}
	for attr := range subsearches {
		var subsearch map[string]any
		subsearch, err = NormalizeCommonLdapSearch(subsearches[attr])
		if err != nil {
			return
		}
		subsearches[attr] = subsearch
		err = normalize.SpuriousKeys(subsearch, "filter", "scope")
		if err != nil {
			return
		}
	}
	return
}

func NormalizeCommonLdapSearch(yaml any) (search map[string]any, err error) {
	search = map[string]any{
		"filter": "(objectClass=*)",
		"scope":  "sub",
	}
	yamlMap, ok := yaml.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("bad type: %T", yaml)
	}
	maps.Copy(search, yamlMap)
	search["filter"] = ldap.CleanFilter(search["filter"].(string))
	return
}
