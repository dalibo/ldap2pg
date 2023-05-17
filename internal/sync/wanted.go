// Logic to describe wanted state from YAML and LDAP
package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleMap
}

func (syncMap Map) Wanted(watch *perf.StopWatch, blacklist lists.Blacklist) (wanted Wanted, err error) {
	var errList []error
	var ldapc ldap.Client
	if syncMap.HasLDAPSearches() {
		ldapOptions, err := ldap.Initialize()
		if err != nil {
			return wanted, err
		}

		ldapc, err = ldap.Connect(ldapOptions)
		if err != nil {
			return wanted, err
		}
		defer ldapc.Conn.Close()
	}

	wanted.Roles = make(map[string]roles.Role)
	for _, item := range syncMap {
		if item.Description != "" {
			slog.Info(item.Description)
		} else {
			slog.Debug("Next sync map item.")
		}

		for data := range item.Search(ldapc, watch) {
			err, failed := data.(error)
			if failed {
				slog.Error("Search error. Keep going.", "err", err)
				errList = append(errList, err)
				continue
			}

			for role := range generateAllRoles(item.RoleRules, data.(*ldap.Results)) {
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
					"name", role.Name, "options", role.Options, "parents", role.Parents, "comment", role.Comment)
				wanted.Roles[role.Name] = role
			}
		}
	}
	if 0 < len(errList) {
		err = errors.Join(errList...)
	}
	return
}

func (wanted Wanted) Sync(watch *perf.StopWatch, real bool, instance inspect.Instance) (count int, err error) {
	ctx := context.Background()
	pool := postgres.DBPool{}
	formatter := postgres.FmtQueryRewriter{}
	defer pool.CloseAll()

	prefix := ""
	if !real {
		prefix = "Would "
	}

	for query := range wanted.diff(instance) {
		slog.Log(ctx, internal.LevelChange, prefix+query.Description, query.LogArgs...)
		count++
		if "" == query.Database {
			query.Database = instance.DefaultDatabase
		}
		pgConn, err := pool.Get(query.Database)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}

		// Rewrite query to log a pasteable query even when in Dry mode.
		sql, _, _ := formatter.RewriteQuery(ctx, pgConn, query.Query, query.QueryArgs)
		slog.Debug(prefix + "Execute SQL query:\n" + sql)

		if !real {
			continue
		}

		var tag pgconn.CommandTag
		duration := watch.TimeIt(func() {
			_, err = pgConn.Exec(ctx, sql)
		})
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}
		slog.Debug("Query terminated.", "duration", duration, "rows", tag.RowsAffected())
	}
	return
}

func (wanted Wanted) diff(instance inspect.Instance) <-chan postgres.SyncQuery {
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

func generateAllRoles(rules []RoleRule, results *ldap.Results) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		for _, rule := range rules {
			for role := range rule.Generate(results) {
				ch <- role
			}
		}
	}()
	return ch
}
