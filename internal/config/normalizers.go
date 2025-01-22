// Functions to normalize YAML input before processing into data structure.
package config

import (
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/normalize"
	"golang.org/x/exp/maps"
)

func NormalizeConfigRoot(yaml interface{}) (config map[string]interface{}, err error) {
	config, ok := yaml.(map[string]interface{})
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

	section, ok = config["privileges"]
	if ok {
		privileges, err := NormalizePrivileges(section)
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

func NormalizePostgres(yaml interface{}) error {
	_, ok := yaml.(map[string]interface{})
	if !ok {
		return fmt.Errorf("bad type: %T, must be a map", yaml)
	}
	return nil
}

func NormalizeRules(yaml interface{}) (syncMap []interface{}, err error) {
	rawRules, ok := yaml.([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type: %T, must be a list", yaml)
	}
	for i, rawRule := range rawRules {
		var item interface{}
		item, err = NormalizeWantRule(rawRule)
		if err != nil {
			return syncMap, fmt.Errorf("item[%d]: %w", i, err)
		}
		syncMap = append(syncMap, item)
	}
	return
}

func NormalizeWantRule(yaml interface{}) (rule map[string]interface{}, err error) {
	rule = map[string]interface{}{
		"description": "",
		"ldapsearch":  map[string]interface{}{},
		"roles":       []interface{}{},
		"grants":      []interface{}{},
	}

	yamlMap, ok := yaml.(map[string]interface{})
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
	rules := []interface{}{}
	for i, rawRule := range list {
		var rule map[string]interface{}
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
	rules = []interface{}{}
	for i, rawRule := range list {
		var rule map[string]interface{}
		rule, err = NormalizeGrantRule(rawRule)
		if err != nil {
			return nil, fmt.Errorf("grants[%d]: %w", i, err)
		}
		for _, rule := range DuplicateGrantRules(rule) {
			rules = append(rules, rule)
		}
	}
	rule["grants"] = rules

	err = normalize.SpuriousKeys(rule, "description", "ldapsearch", "roles", "grants")
	return
}

func NormalizeLdapSearch(yaml interface{}) (search map[string]interface{}, err error) {
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
		err = normalize.SpuriousKeys(subsearch, "filter", "scope")
		if err != nil {
			return
		}
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
		return nil, fmt.Errorf("bad type: %T", yaml)
	}
	maps.Copy(search, yamlMap)
	search["filter"] = ldap.CleanFilter(search["filter"].(string))
	return
}
