package states

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/exp/slog"
)

func (instance *PostgresInstance) Diff(wanted Wanted) <-chan postgres.SyncQuery {
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

func (instance *PostgresInstance) Sync(watch *perf.StopWatch, real bool, wanted Wanted) (count int, err error) {
	ctx := context.Background()
	pool := postgres.DBPool{}
	formatter := postgres.FmtQueryRewriter{}
	defer pool.CloseAll()

	prefix := ""
	if !real {
		prefix = "Would "
	}

	for query := range instance.Diff(wanted) {
		slog.Log(ctx, config.LevelChange, prefix+query.Description, query.LogArgs...)
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
