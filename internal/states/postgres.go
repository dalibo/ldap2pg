package states

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

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
	FallbackOwner    string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     roles.RoleMap
	Me               roles.Role
	RolesBlacklist   utils.Blacklist
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
	if "" == c.Postgres.FallbackOwner {
		instance.FallbackOwner = instance.Me.Name
	} else {
		instance.FallbackOwner = c.Postgres.FallbackOwner
	}
	slog.Debug("Fallback owner configured.", "role", instance.FallbackOwner)

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
	slog.Debug("Inspecting roles options.")
	var columns []string
	err := utils.IterateToSlice(postgres.RunQuery(roleColumnsQuery, pgconn, pgx.RowTo[string], nil), &columns)
	if err != nil {
		return err
	}
	// Setup global var to configure RoleOptions.String()
	config.ProcessRoleColumns(columns, instance.Me.Options.Super)
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", columns)

	slog.Debug("Inspecting roles blacklist.")
	for item := range postgres.RunQuery(c.Postgres.RolesBlacklistQuery, pgconn, pgx.RowTo[string], config.YamlToString) {
		if err, _ := item.(error); err != nil {
			return err
		}
		pattern := item.(string)
		instance.RolesBlacklist = append(instance.RolesBlacklist, pattern)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

	instance.AllRoles = make(roles.RoleMap)
	sql := "rol." + strings.Join(columns, ", rol.")
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

func (instance *PostgresInstance) Sync(timer *utils.Timer, real bool, wanted Wanted) (count int, err error) {
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

		timer.TimeIt(func() {
			_, err = pgconn.Exec(ctx, sql)
		})

		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}
	}
	return
}
