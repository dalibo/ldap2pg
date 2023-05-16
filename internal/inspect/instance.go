package inspect

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type Instance struct {
	AllRoles         roles.RoleMap
	Databases        []postgres.Database
	DefaultDatabase  string
	FallbackOwner    string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     roles.RoleMap
	Me               roles.Role
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

func InstanceState(pc Config) (instance Instance, err error) {
	instance = Instance{}
	instance.ManagedDatabases = mapset.NewSet[string]()

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	err = instance.InspectSession(pc, pgconn)
	if err != nil {
		return
	}
	err = instance.InspectDatabases(pc.DatabasesQuery, pgconn)
	if err != nil {
		return instance, fmt.Errorf("postgres: %w", err)
	}

	err = instance.InspectRoles(pgconn, pc.RolesBlacklistQuery, pc.ManagedRolesQuery)
	if err != nil {
		return
	}
	return
}

func (instance *Instance) InspectSession(pc Config, pgconn *pgx.Conn) error {
	slog.Debug("Inspecting PostgreSQL server and session.")
	slog.Debug("Executing SQL query:\n" + sessionQuery)
	rows, err := pgconn.Query(context.Background(), sessionQuery)
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

func (instance *Instance) InspectDatabases(q Querier[string], pgconn *pgx.Conn) error {
	slog.Debug("Inspecting managed databases.")
	for q.Query(pgconn); q.Next(); {
		instance.ManagedDatabases.Add(q.Row())
	}
	if err := q.Err(); err != nil {
		return fmt.Errorf("databases: %w", err)
	}

	slog.Debug("Inspecting database owners.")
	for item := range RunQuery(databasesQuery, pgconn, postgres.RowToDatabase, nil) {
		err, _ := item.(error)
		if err != nil {
			return err
		}
		db := item.(postgres.Database)
		if instance.ManagedDatabases.Contains(db.Name) {
			slog.Debug("Found database.", "name", db.Name)
			instance.Databases = append(instance.Databases, db)
		}
	}
	return nil
}

func (instance *Instance) InspectRoles(pgconn *pgx.Conn, rolesBlackListQ Querier[string], managedRolesQ RowsOrSQL) error {
	slog.Debug("Inspecting roles options.")
	var columns []string
	err := lists.IterateToSlice(RunQuery(roleColumnsQuery, pgconn, pgx.RowTo[string], nil), &columns)
	if err != nil {
		return err
	}
	// Setup global var to configure RoleOptions.String()
	roles.ProcessColumns(columns, instance.Me.Options.Super)
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", columns)

	slog.Debug("Inspecting roles blacklist.")
	for rolesBlackListQ.Query(pgconn); rolesBlackListQ.Next(); {
		instance.RolesBlacklist = append(instance.RolesBlacklist, rolesBlackListQ.Row())
	}
	if err := rolesBlackListQ.Err(); err != nil {
		return fmt.Errorf("roles_blacklist_query: %w", err)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

	instance.AllRoles = make(roles.RoleMap)
	sql := "rol." + strings.Join(columns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	slog.Debug("Inspecting all roles.")
	for item := range RunQuery(rolesQuery, pgconn, roles.RowToRole, nil) {
		if err, _ := item.(error); err != nil {
			return err
		}
		role := item.(roles.Role)
		match := instance.RolesBlacklist.Match(&role)
		if match == "" {
			instance.AllRoles[role.Name] = role
			slog.Debug("Found role in Postgres instance.", "name", role.Name, "options", role.Options, "parents", role.Parents)

		} else {
			slog.Debug("Ignoring blacklisted role name.", "name", role.Name, "pattern", match)
		}
	}
	if nil == managedRolesQ.Value {
		slog.Debug("Managing all roles found.")
		instance.ManagedRoles = instance.AllRoles
		return nil
	}

	slog.Debug("Inspecting managed roles.")
	instance.ManagedRoles = make(roles.RoleMap)
	for item := range RunQuery(managedRolesQ, pgconn, pgx.RowTo[string], YamlToString) {
		if err, _ := item.(error); err != nil {
			return err
		}
		name := item.(string)
		match := instance.RolesBlacklist.MatchString(name)
		if match == "" {
			instance.ManagedRoles[name] = instance.AllRoles[name]
			slog.Debug("Managing role.", "role", name)

		} else {
			slog.Debug("Ignoring blacklisted role name.", "role", name, "pattern", match)
		}
	}
	return nil
}
