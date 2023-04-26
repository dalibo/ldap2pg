// Logic to describe wanted state from YAML and LDAP
package states

import (
	"context"
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	mapset "github.com/deckarep/golang-set/v2"
	ldapv3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleMap
}

func ComputeWanted(config config.Config) (wanted Wanted, err error) {
	var ldapConn *ldapv3.Conn
	if config.HasLDAPSearches() {
		ldapOptions, err := ldap.Initialize()
		if err != nil {
			return wanted, err
		}

		ldapConn, err = ldap.Connect(ldapOptions)
		if err != nil {
			return wanted, err
		}
		defer ldapConn.Close()
	}

	wanted.Roles = make(map[string]roles.Role)
	for _, item := range config.SyncItems {
		var entries []*ldapv3.Entry
		if item.Description != "" {
			slog.Info(item.Description)
		}

		if item.HasLDAPSearch() {
			search := ldapv3.SearchRequest{
				BaseDN:     item.LdapSearch.Base,
				Scope:      ldapv3.ScopeWholeSubtree,
				Filter:     ldap.CleanFilter(item.LdapSearch.Filter),
				Attributes: item.LdapSearch.Attributes,
			}
			slog.Debug("Searching LDAP directory.", "base", search.BaseDN, "filter", search.Filter, "attributes", search.Attributes)
			res, err := ldapConn.Search(&search)
			if err != nil {
				return wanted, err
			}
			entries = res.Entries
		} else {
			entries = [](*ldapv3.Entry){nil}
		}

		for _, entry := range entries {
			if entry != nil {
				slog.Debug("Got LDAP entry.", "dn", entry.DN)
				continue
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
					slog.Debug("Wants role.", "name", role.Name, "options", role.Options, "parents", role.Parents)
					wanted.Roles[role.Name] = role
				}
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
		case 1: // Copy same comment for all roles.
		default:
			if commentsLen != len(rule.Names) {
				ch <- interface{}(errors.New("Comment list inconsistent with generated names"))
				return
			}
		}
		var comments []string
		for _, comment := range rule.Comments {
			comments = append(comments, comment.String())
		}

		var parents []string
		for _, parent := range rule.Parents {
			parents = append(parents, parent.String())
		}

		for i, name := range rule.Names {
			role := roles.NewRole()
			role.Name = name.String()
			role.Options = rule.Options
			role.Parents = mapset.NewSet[string](parents...)
			if 1 == commentsLen {
				role.Comment = comments[0]
			} else {
				role.Comment = comments[i]
			}

			ch <- role
		}
	}()
	return ch
}

func (wanted *Wanted) Diff(instance PostgresInstance) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing.
		for _, name := range wanted.Roles.Flatten() {
			role := wanted.Roles[name]
			if other, ok := instance.AllRoles[name]; ok {
				// Check for existing role, even if unmanaged.
				if _, ok := instance.ManagedRoles[name]; !ok {
					slog.Warn("Reusing unmanaged role. Ensure managed_roles_query returns all wanted roles.", "role", name)
				}
				other.Alter(role, ch)
			} else {
				role.Create(ch)
			}
		}

		// Drop spurious.
		// Only from managed roles.
		for name := range instance.ManagedRoles {
			if _, ok := wanted.Roles[name]; ok {
				continue
			}

			if "public" == name {
				continue
			}

			role, ok := instance.AllRoles[name]
			if !ok {
				// Already dropped. ldap2pg hits this case whan
				// ManagedRoles is static.
				continue
			}

			role.Drop(instance.Databases, instance.Me, instance.FallbackOwner, ch)
		}
	}()
	return ch
}

func (wanted *Wanted) Sync(real bool, c config.Config, instance PostgresInstance) (count int, err error) {
	ctx := context.Background()
	pool := postgres.DBPool{}
	formatter := postgres.FmtQueryRewriter{}
	defer pool.CloseAll()

	prefix := ""
	if !real {
		prefix = "Would "
	}

	for query := range wanted.Diff(instance) {
		slog.Info(prefix+query.Description, query.LogArgs...)
		count++
		if "" == query.Database {
			query.Database = instance.DefaultDatabase
		}
		pgconn, err := pool.Get(query.Database)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}

		// Rewrite query to log a pasteable query even when in Dry mode.
		sql, _, _ := formatter.RewriteQuery(ctx, pgconn, query.Query, query.QueryArgs)
		slog.Debug(prefix + "Execute SQL query:\n" + sql)

		if !real {
			continue
		}

		_, err = pgconn.Exec(ctx, sql)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}
	}
	return
}
