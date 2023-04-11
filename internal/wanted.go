// Logic to describe wanted state from YAML and LDAP
package internal

import (
	"errors"
	"fmt"

	"golang.org/x/exp/slog"
)

type WantedState struct {
	Roles RoleSet
}

type SyncItem struct {
	Description string
	LdapSearch  interface{}
	RoleRules   []RoleRule
}

type RoleRule struct {
	Names    []string
	Comments []string
}

func (item *SyncItem) LoadYaml(yaml map[string]interface{}) (err error) {
	desc, ok := yaml["description"]
	if ok {
		item.Description = desc.(string)
	}
	rules, ok := yaml["roles"]
	if ok {
		ruleList := rules.([]interface{})
		for _, yamlRule := range ruleList {
			rule := RoleRule{}
			yamlRuleMap := yamlRule.(map[string]interface{})
			rule.LoadYaml(yamlRuleMap)
			item.RoleRules = append(item.RoleRules, rule)
		}
	}
	iLdap, exists := yaml["ldapsearch"]
	if exists {
		item.LdapSearch = iLdap
	}
	return
}

func (rule *RoleRule) LoadYaml(yaml map[string]interface{}) {
	rule.Names = yaml["names"].([]string)
	rule.Comments = yaml["comments"].([]string)
}

func (rule *RoleRule) Generate() (roles []Role, err error) {
	commentsLen := len(rule.Comments)
	switch commentsLen {
	case 0:
		rule.Comments = []string{"Managed by ldap2pg"}
		commentsLen = 1
	case 1: // Copy same comment for all roles.
	default:
		if commentsLen != len(rule.Names) {
			err = errors.New("Comment list inconsistent with generated names")
			return
		}
	}

	for i, name := range rule.Names {
		role := Role{Name: name}
		if 1 == commentsLen {
			role.Comment = rule.Comments[0]
		} else {
			role.Comment = rule.Comments[i]
		}
		roles = append(roles, role)
	}
	return
}

func ComputeWanted(config Config) (wanted WantedState, err error) {
	wanted.Roles = make(map[string]Role)
	for _, item := range config.SyncMap {
		if item.LdapSearch != nil {
			slog.Debug("Skipping LDAP search for now.",
				"description", item.Description)

			continue
		}
		if item.Description != "" {
			slog.Info(item.Description)
		}

		for _, rule := range item.RoleRules {
			var roles []Role
			roles, err = rule.Generate()
			if err != nil {
				return
			}

			for _, role := range roles {
				_, exists := wanted.Roles[role.Name]
				if exists {
					err = fmt.Errorf("Duplicated role %s", role.Name)
					return
				}
				slog.Debug("Wants role.", "name", role.Name)
				wanted.Roles[role.Name] = role
			}
		}
	}
	return
}
