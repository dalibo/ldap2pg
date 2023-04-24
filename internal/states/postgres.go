package states

import (
	"context"
	_ "embed"
	"strings"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/dalibo/ldap2pg/internal/utils"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type PostgresInstance struct {
	AllRoles         roles.RoleMap
	Databases        []postgres.Database
	DefaultDatabase  string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     roles.RoleMap
	Me               roles.Role
	RoleColumns      []string
	RolesBlacklist   utils.Blacklist
	ServerVersion    string
	ServerVersionNum int
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

func PostgresInspect(c config.Config) (instance PostgresInstance, err error) {
	instance = PostgresInstance{}
	instance.ManagedDatabases = mapset.NewSet[string]()

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	err = instance.InspectSession(c, pgconn)
	if err != nil {
		return
	}
	err = instance.InspectDatabases(c, pgconn)
	if err != nil {
		return
	}

	err = instance.InspectRoles(c, pgconn)
	if err != nil {
		return
	}
	return
}

func (instance *PostgresInstance) InspectSession(c config.Config, pgconn *pgx.Conn) error {
	slog.Debug("Inspecting PostgreSQL server and session.")
	slog.Debug("Executing SQL query:\n" + sessionQuery)
	rows, err := pgconn.Query(context.Background(), sessionQuery)
	if err != nil {
		return err
	}
	if !rows.Next() {
		panic("No data returned.")
	}
	var clusterName string
	err = rows.Scan(
		&instance.ServerVersion, &instance.ServerVersionNum,
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
		"version", instance.ServerVersion,
		"cluster", clusterName,
		"db", instance.DefaultDatabase,
	)
	if rows.Next() {
		panic("Multiple row returned.")
	}
	return nil
}

func (instance *PostgresInstance) InspectDatabases(c config.Config, pgconn *pgx.Conn) error {
	slog.Debug("Inspecting managed databases.")
	err := utils.IterateToSet(postgres.RunQuery(c.Postgres.DatabasesQuery, pgconn, pgx.RowTo[string], config.YamlToString), &instance.ManagedDatabases)
	if err != nil {
		return err
	}
	slog.Debug("Inspecting database owners.")
	for item := range postgres.RunQuery(databasesQuery, pgconn, postgres.RowToDatabase, nil) {
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

func (instance *PostgresInstance) InspectRoles(c config.Config, pgconn *pgx.Conn) error {
	slog.Debug("Inspecting roles blacklist.")
	for item := range postgres.RunQuery(c.Postgres.RolesBlacklistQuery, pgconn, pgx.RowTo[string], config.YamlToString) {
		if err, _ := item.(error); err != nil {
			return err
		}
		pattern := item.(string)
		instance.RolesBlacklist = append(instance.RolesBlacklist, pattern)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

	slog.Debug("Inspecting roles options.")
	err := utils.IterateToSlice(postgres.RunQuery(roleColumnsQuery, pgconn, pgx.RowTo[string], nil), &instance.RoleColumns)
	if err != nil {
		return err
	}
	// Setup global var to configure RoleOptions.String()
	config.ProcessRoleColumns(instance.RoleColumns, instance.Me.Options.Super)
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", instance.RoleColumns)

	instance.AllRoles = make(roles.RoleMap)
	sql := "rol." + strings.Join(instance.RoleColumns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	slog.Debug("Inspecting all roles.")
	for item := range postgres.RunQuery(rolesQuery, pgconn, roles.RowToRole, nil) {
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
	if nil == c.Postgres.ManagedRolesQuery {
		slog.Debug("Managing all roles found.")
		instance.ManagedRoles = instance.AllRoles
		return nil
	}

	slog.Debug("Inspecting managed roles.")
	instance.ManagedRoles = make(roles.RoleMap)
	for item := range postgres.RunQuery(c.Postgres.ManagedRolesQuery, pgconn, pgx.RowTo[string], config.YamlToString) {
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
