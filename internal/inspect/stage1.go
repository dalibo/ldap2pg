package inspect

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

var (
	//go:embed sql/databases.sql
	databasesQuery string
	//go:embed sql/role-columns.sql
	roleColumnsQuery string
	//go:embed sql/roles.sql
	rolesQuery string
	//go:embed sql/session.sql
	sessionQuery string
)

func (instance *Instance) InspectStage1(ctx context.Context, pc Config) (err error) {
	slog.Debug("Stage 1: roles.")
	instance.ManagedDatabases = mapset.NewSet[string]()

	pgconn, err := postgres.GetConn(ctx, "")
	if err != nil {
		return
	}

	err = instance.InspectManagedDatabases(ctx, pgconn, pc.DatabasesQuery)
	if err != nil {
		return fmt.Errorf("databases: %w", err)
	}

	err = instance.InspectRoles(ctx, pgconn, pc.ManagedRolesQuery)
	if err != nil {
		return fmt.Errorf("roles: %w", err)
	}
	return
}

func (instance *Instance) InspectManagedDatabases(ctx context.Context, pgconn *pgx.Conn, q Querier[string]) error {
	slog.Debug("Inspecting managed databases.")
	for q.Query(ctx, pgconn); q.Next(); {
		instance.ManagedDatabases.Add(q.Row())
	}
	if err := q.Err(); err != nil {
		return err
	}

	slog.Debug("Inspecting database owners.")
	instance.Databases = make(postgres.DBMap)
	dbq := &SQLQuery[postgres.Database]{SQL: databasesQuery, RowTo: postgres.RowToDatabase}
	for dbq.Query(ctx, pgconn); dbq.Next(); {
		db := dbq.Row()
		if instance.ManagedDatabases.Contains(db.Name) {
			slog.Debug("Found database.", "name", db.Name)
			instance.Databases[db.Name] = db
		}
	}
	if err := dbq.Err(); err != nil {
		return err
	}

	_, ok := instance.Databases[instance.DefaultDatabase]
	if !ok {
		return fmt.Errorf("default database not listed")
	}
	return nil
}

func (instance *Instance) InspectRoles(ctx context.Context, pgconn *pgx.Conn, managedRolesQ Querier[string]) error {
	slog.Debug("Inspecting roles options.")
	var columns []string
	q := &SQLQuery[string]{SQL: roleColumnsQuery, RowTo: pgx.RowTo[string]}
	for q.Query(ctx, pgconn); q.Next(); {
		columns = append(columns, q.Row())
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("columns: %w", err)
	}
	// Setup global var to configure RoleOptions.String()
	role.ProcessColumns(columns, instance.Me.Options.Super)
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", columns)

	slog.Debug("Inspecting all roles.")
	instance.AllRoles = make(role.Map)
	sql := "rol." + strings.Join(columns, ", rol.")
	sql = strings.Replace(rolesQuery, "rol.*", sql, 1)
	rq := &SQLQuery[role.Role]{SQL: sql, RowTo: role.RowTo}
	for rq.Query(ctx, pgconn); rq.Next(); {
		role := rq.Row()
		match := instance.RolesBlacklist.Match(&role)
		if match == "" {
			instance.AllRoles[role.Name] = role
			slog.Debug("Found role in Postgres instance.", "name", role.Name, "options", role.Options, "parents", role.Parents.ToSlice())

		} else {
			slog.Debug("Ignoring blacklisted role name.", "name", role.Name, "pattern", match)
		}
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("options: %w", err)
	}

	if nil == managedRolesQ {
		slog.Debug("Managing all roles found.")
		instance.ManagedRoles = instance.AllRoles
		return nil
	}

	slog.Debug("Inspecting managed roles.")
	instance.ManagedRoles = make(role.Map)
	for managedRolesQ.Query(ctx, pgconn); managedRolesQ.Next(); {
		name := managedRolesQ.Row()
		match := instance.RolesBlacklist.MatchString(name)
		if "" != match {
			slog.Debug("Ignoring blacklisted role name.", "role", name, "pattern", match)
			continue
		}
		instance.ManagedRoles[name] = instance.AllRoles[name]
		slog.Debug("Managing role.", "role", name)

	}
	if err := managedRolesQ.Err(); err != nil {
		return fmt.Errorf("managed_roles_query: %w", err)
	}

	return nil
}
