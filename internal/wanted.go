// Logic to describe wanted state from YAML and LDAP
package internal

import (
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"golang.org/x/exp/slog"
)

type WantedState struct {
	Roles roles.RoleSet
}

func ComputeWanted(config config.Config) (wanted WantedState, err error) {
	wanted.Roles = make(map[string]roles.Role)
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
			var roleList []roles.Role
			roleList, err = GenerateRoles(rule)
			if err != nil {
				return
			}

			for _, role := range roleList {
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

func GenerateRoles(rule config.RoleRule) (roleList []roles.Role, err error) {
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
		role := roles.Role{Name: name}
		if 1 == commentsLen {
			role.Comment = rule.Comments[0]
		} else {
			role.Comment = rule.Comments[i]
		}
		roleList = append(roleList, role)
	}
	return
}

func (wanted *WantedState) Diff(instance PostgresInstance) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing
		for name := range wanted.Roles {
			if _, ok := instance.AllRoles[name]; ok {
				slog.Debug("Role already in instance.", "role", name)
				continue
			}

			role := wanted.Roles[name]
			role.Create(ch)
		}

		// Drop spurious
		for name := range instance.ManagedRoles {
			if _, ok := wanted.Roles[name]; ok {
				continue
			}

			if "public" == name {
				continue
			}

			role := instance.ManagedRoles[name]
			role.Drop(instance.Databases, ch)
		}
	}()
	return ch
}
