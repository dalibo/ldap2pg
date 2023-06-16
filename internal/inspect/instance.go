package inspect

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type Instance struct {
	AllRoles         role.Map
	Databases        []postgres.Database
	DefaultDatabase  string
	FallbackOwner    string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     role.Map
	Me               role.Role
	RolesBlacklist   lists.Blacklist
}

var (
	//go:embed sql/inspect-databases.sql
	databasesQuery string
	//go:embed sql/role-columns.sql
	roleColumnsQuery string
	//go:embed sql/roles.sql
	rolesQuery string
	//go:embed sql/session.sql
	sessionQuery string
)

func (pc Config) InspectStage1(ctx context.Context) (instance Instance, err error) {
	instance = Instance{
		ManagedDatabases: mapset.NewSet[string](),
	}

	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	err = instance.InspectSession(ctx, pgconn, pc)
	if err != nil {
		return
	}
	err = instance.InspectDatabases(ctx, pgconn, pc.DatabasesQuery)
	if err != nil {
		return instance, fmt.Errorf("postgres: %w", err)
	}

	err = instance.InspectRoles(ctx, pgconn, pc.RolesBlacklistQuery, pc.ManagedRolesQuery)
	if err != nil {
		return
	}
	return
}

func (instance *Instance) InspectSession(ctx context.Context, pgconn *pgx.Conn, pc Config) error {
	slog.Debug("Inspecting PostgreSQL server and session.")
	slog.Debug("Executing SQL query:\n" + sessionQuery)
	rows, err := pgconn.Query(ctx, sessionQuery)
	if err != nil {
		return err
	}
	if !rows.Next() {
		panic("No data returned.")
	}
	var clusterName, serverVersion string
	var serverVersionNum int
	err = rows.Scan(
		&serverVersion, &serverVersionNum,
		&clusterName, &instance.DefaultDatabase,
		&instance.Me.Name, &instance.Me.Options.Super,
	)
	if err != nil {
		return err
	}
	var msg string
	if instance.Me.Options.Super {
		msg = "Running as superuser."
	} else {
		msg = "Running as unprivileged user."
	}
	slog.Info(
		msg,
		"user", instance.Me.Name,
		"super", instance.Me.Options.Super,
		"version", serverVersion,
		"cluster", clusterName,
		"db", instance.DefaultDatabase,
	)
	if rows.Next() {
		panic("Multiple row returned.")
	}
	if "" == pc.FallbackOwner {
		instance.FallbackOwner = instance.Me.Name
	} else {
		instance.FallbackOwner = pc.FallbackOwner
	}
	slog.Debug("Fallback owner configured.", "role", instance.FallbackOwner)

	return nil
}

func (instance *Instance) InspectDatabases(ctx context.Context, pgconn *pgx.Conn, q Querier[string]) error {
	slog.Debug("Inspecting managed databases.")
	for q.Query(ctx, pgconn); q.Next(); {
		instance.ManagedDatabases.Add(q.Row())
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("databases: %w", err)
	}

	slog.Debug("Inspecting database owners.")
	dbq := &SQLQuery[postgres.Database]{SQL: databasesQuery, RowTo: postgres.RowToDatabase}
	for dbq.Query(ctx, pgconn); dbq.Next(); {
		db := dbq.Row()
		if instance.ManagedDatabases.Contains(db.Name) {
			slog.Debug("Found database.", "name", db.Name)
			instance.Databases = append(instance.Databases, db)
		}
	}
	if err := dbq.Err(); err != nil {
		return fmt.Errorf("databases: %w", err)
	}

	return nil
}

func (instance *Instance) InspectRoles(ctx context.Context, pgconn *pgx.Conn, rolesBlackListQ, managedRolesQ Querier[string]) error {
	slog.Debug("Inspecting roles options.")
	var columns []string
	q := &SQLQuery[string]{SQL: roleColumnsQuery, RowTo: pgx.RowTo[string]}
	for q.Query(ctx, pgconn); q.Next(); {
		columns = append(columns, q.Row())
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("role columns: %w", err)
	}
	// Setup global var to configure RoleOptions.String()
	role.ProcessColumns(columns, instance.Me.Options.Super)
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", columns)

	slog.Debug("Inspecting roles blacklist.")
	for rolesBlackListQ.Query(ctx, pgconn); rolesBlackListQ.Next(); {
		instance.RolesBlacklist = append(instance.RolesBlacklist, rolesBlackListQ.Row())
	}
	if err := rolesBlackListQ.Err(); err != nil {
		return fmt.Errorf("roles_blacklist_query: %w", err)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

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
			slog.Debug("Found role in Postgres instance.", "name", role.Name, "options", role.Options, "parents", role.Parents)

		} else {
			slog.Debug("Ignoring blacklisted role name.", "name", role.Name, "pattern", match)
		}
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("roles options: %w", err)
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
