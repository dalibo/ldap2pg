package sync

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/exp/slog"
)

type RoleRule struct {
	Name    pyfmt.Format
	Options role.Options
	Comment pyfmt.Format
	Parents []pyfmt.Format
}

func (r RoleRule) IsStatic() bool {
	return r.Name.IsStatic() &&
		r.Comment.IsStatic() &&
		lists.And(r.Parents, func(f pyfmt.Format) bool { return f.IsStatic() })
}

func (r RoleRule) Generate(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
	go func() {
		defer close(ch)
		parents := mapset.NewSet[string]()
		for _, f := range r.Parents {
			if nil == results.Entry || 0 == len(f.Fields) {
				// Static case.
				parents.Add(f.String())
			} else {
				// Dynamic case.
				for values := range results.GenerateValues(f) {
					parents.Add(f.Format(values))
				}
			}
		}

		if nil == results.Entry {
			// Case static rule.
			role := role.Role{
				Name:    r.Name.String(),
				Comment: r.Comment.String(),
				Options: r.Options,
				Parents: parents,
			}
			ch <- role
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Name, r.Comment) {
				role := role.Role{}
				role.Name = r.Name.Format(values)
				role.Comment = r.Comment.Format(values)
				role.Options = r.Options
				role.Parents = parents.Clone()
				ch <- role
			}
		}
	}()
	return ch
}

func (wanted Wanted) Sync(ctx context.Context, watch *perf.StopWatch, real bool, ch <-chan postgres.SyncQuery) (count int, err error) {
	pool := postgres.DBPool{}
	formatter := postgres.FmtQueryRewriter{}
	defer pool.CloseAll(ctx)

	prefix := ""
	if !real {
		prefix = "Would "
	}

	for query := range ch {
		slog.Log(ctx, internal.LevelChange, prefix+query.Description, query.LogArgs...)
		count++
		pgConn, err := pool.Get(ctx, query.Database)
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
			return count, fmt.Errorf("sync: %w", err)
		}
		slog.Debug("Query terminated.", "duration", duration, "rows", tag.RowsAffected())
	}
	return
}

func (wanted Wanted) DiffRoles(instance inspect.Instance) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing roles.
		for _, name := range wanted.Roles.Flatten() {
			role := wanted.Roles[name]
			if other, ok := instance.AllRoles[name]; ok {
				// Check for existing role, even if unmanaged.
				if _, ok := instance.ManagedRoles[name]; !ok {
					slog.Warn("Reusing unmanaged role. Ensure managed_roles_query returns all wanted roles.", "role", name)
				}
				sendQueries(other.Alter(role), ch, instance.DefaultDatabase)
			} else {
				sendQueries(role.Create(), ch, instance.DefaultDatabase)
			}
		}

		// Drop spurious roles.
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

			sendQueries(role.Drop(instance.Databases, instance.Me, instance.FallbackOwner), ch, instance.DefaultDatabase)
		}
	}()
	return ch
}

func sendQueries(queries []postgres.SyncQuery, ch chan postgres.SyncQuery, defaultDatabase string) {
	for _, q := range queries {
		if "" == q.Database {
			q.Database = defaultDatabase
		}
		ch <- q
	}
}
