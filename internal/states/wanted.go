// Logic to describe wanted state from YAML and LDAP
package states

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/dalibo/ldap2pg/internal/utils"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleMap
}

func ComputeWanted(config config.Config, blacklist utils.Blacklist) (wanted Wanted, err error) {
	var ldapConn *ldap3.Conn
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
		var entries []*ldap3.Entry
		if item.Description != "" {
			slog.Info(item.Description)
		}

		if item.HasLDAPSearch() {
			search := ldap3.SearchRequest{
				BaseDN:     item.LdapSearch.Base,
				Scope:      ldap3.ScopeWholeSubtree,
				Filter:     ldap.CleanFilter(item.LdapSearch.Filter),
				Attributes: item.LdapSearch.Attributes,
			}
			slog.Debug("Searching LDAP directory.",
				"base", search.BaseDN, "filter", search.Filter, "attributes", search.Attributes)
			res, err := ldapConn.Search(&search)
			if err != nil {
				return wanted, err
			}
			entries = res.Entries
		} else {
			entries = [](*ldap3.Entry){nil}
		}

		for _, rule := range item.RoleRules {
			for _, entry := range entries {
				for role := range GenerateRoles(rule, entry) {
					if "" == role.Name {
						continue
					}
					pattern := blacklist.MatchString(role.Name)
					if pattern != "" {
						slog.Debug(
							"Ignoring blacklisted wanted role.",
							"role", role.Name, "pattern", pattern)
						continue
					}
					_, exists := wanted.Roles[role.Name]
					if exists {
						slog.Warn("Duplicated wanted role.", "role", role.Name)
					}
					slog.Debug("Wants role.",
						"name", role.Name, "options", role.Options, "parents", role.Parents)
					wanted.Roles[role.Name] = role
				}
			}
		}
	}
	return
}

func GenerateRoles(rule config.RoleRule, entry *ldap3.Entry) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		var parents []string
		for _, f := range rule.Parents {
			if nil == entry || 0 == len(f.Fields) {
				// Static case.
				parents = append(parents, f.String())
			} else {
				// Dynamic case.
				for values := range ldap.GenerateValues(entry, f) {
					parents = append(parents, f.Format(values))
				}
			}
		}

		if nil == entry {
			// Case static role.
			role := roles.Role{}
			role.Name = rule.Name.String()
			role.Comment = rule.Comment.String()
			role.Options = rule.Options
			role.Parents = mapset.NewSet[string](parents...)
			ch <- role
		} else {
			// Case dynamic roles.
			for values := range ldap.GenerateValues(entry, rule.Name, rule.Comment) {
				role := roles.Role{}
				role.Name = rule.Name.Format(values)
				role.Comment = rule.Comment.Format(values)
				role.Options = rule.Options
				role.Parents = mapset.NewSet[string](parents...)
				ch <- role
			}
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
