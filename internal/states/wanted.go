// Logic to describe wanted state from YAML and LDAP
package states

import (
	"context"
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleSet
}

func ComputeWanted(config config.Config) (wanted Wanted, err error) {
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
			for item := range GenerateRoles(rule) {
				err, _ := item.(error)
				if err != nil {
					return wanted, err
				}
				role, ok := item.(roles.Role)
				if !ok {
					panic(fmt.Sprintf("bad object generated: %v", item))
				}
				_, exists := wanted.Roles[role.Name]
				if exists {
					err = fmt.Errorf("Duplicated role %s", role.Name)
					return wanted, err
				}
				slog.Debug("Wants role.", "name", role.Name, "options", role.Options)
				wanted.Roles[role.Name] = role
			}
		}
	}
	return
}

func GenerateRoles(rule config.RoleRule) (ch chan interface{}) {
	ch = make(chan interface{})
	go func() {
		defer close(ch)
		commentsLen := len(rule.Comments)
		switch commentsLen {
		case 0:
			rule.Comments = []string{"Managed by ldap2pg"}
			commentsLen = 1
		case 1: // Copy same comment for all roles.
		default:
			if commentsLen != len(rule.Names) {
				ch <- interface{}(errors.New("Comment list inconsistent with generated names"))
				return
			}
		}

		for i, name := range rule.Names {
			role := roles.Role{Name: name, Options: rule.Options}
			if 1 == commentsLen {
				role.Comment = rule.Comments[0]
			} else {
				role.Comment = rule.Comments[i]
			}
			ch <- interface{}(role)
		}
	}()
	return ch
}

func (wanted *Wanted) Diff(instance PostgresInstance) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing
		for name := range wanted.Roles {
			role := wanted.Roles[name]
			if other, ok := instance.AllRoles[name]; ok {
				other.Alter(role, ch)
			} else {
				role.Create(ch)
			}
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

func (wanted *Wanted) Sync(c config.Config, instance PostgresInstance) (count int, err error) {
	ctx := context.Background()
	pool := postgres.DBPool{}
	defer pool.CloseAll()

	prefix := ""
	if c.Dry {
		prefix = "Would "
	}

	for query := range wanted.Diff(instance) {
		slog.Info(prefix+query.Description, query.LogArgs...)
		slog.Debug(prefix+"Execute SQL query:\n"+query.Query, "args", query.QueryArgs)
		count++
		if c.Dry {
			continue
		}

		pgconn, err := pool.Get(query.Database)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}
		_, err = pgconn.Exec(ctx, query.Query, query.QueryArgs...)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}
	}
	return
}
