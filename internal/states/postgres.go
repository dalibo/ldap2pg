package states

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/dalibo/ldap2pg/internal/utils"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type PostgresInstance struct {
	AllRoles       roles.RoleSet
	Databases      []string
	ManagedRoles   roles.RoleSet
	RoleColumns    []string
	RolesBlacklist utils.Blacklist
}

var (
	//go:embed sql/role-columns.sql
	roleColumnsQuery string
	//go:embed sql/roles.sql
	rolesQuery string
)

func PostgresInspect(config config.Config) (instance PostgresInstance, err error) {
	instance = PostgresInstance{}

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	instance.Databases, err = postgres.RunQuery(config.Postgres.DatabasesQuery, pgconn, postgres.RowToString, postgres.YamlToString)
	if err != nil {
		return
	}
	for _, name := range instance.Databases {
		slog.Debug("Found database.", "name", name)
	}

	patterns, err := postgres.RunQuery(config.Postgres.RolesBlacklistQuery, pgconn, postgres.RowToString, postgres.YamlToString)
	if err != nil {
		return
	}
	instance.RolesBlacklist = utils.Blacklist(patterns)

	rows, err := pgconn.Query(ctx, roleColumnsQuery)
	slog.Debug(roleColumnsQuery)
	if err != nil {
		slog.Error("Failed to query role columns.")
		return
	}
	columns, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		err = fmt.Errorf("Failed to fetch rows: %w", err)
		return
	}
	instance.RoleColumns = columns
	slog.Debug("Querying PostgreSQL instance role columns.", "columns", instance.RoleColumns)

	sql := "rol." + strings.Join(instance.RoleColumns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	slog.Debug(rolesQuery)
	rows, err = pgconn.Query(ctx, rolesQuery)
	if err != nil {
		err = fmt.Errorf("Failed to query role columns: %s", err)
		return
	}
	unfilteredRoles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (role roles.Role, err error) {
		role, err = roles.NewRoleFromRow(row, instance.RoleColumns)
		return
	})
	if err != nil {
		err = fmt.Errorf("Failed to fetch rows: %w", err)
		return
	}

	instance.AllRoles = make(roles.RoleSet)
	for _, role := range unfilteredRoles {
		match := instance.RolesBlacklist.Match(&role)
		if match == "" {
			instance.AllRoles[role.Name] = role
			slog.Debug("Found role in Postgres instance.", "name", role.Name, "super", role.Options.Super)

		} else {
			slog.Debug("Ignoring blacklisted role name.", "name", role.Name, "pattern", match)
		}
	}

	err = instance.InspectManagedRoles(config, pgconn)
	return
}

func (instance *PostgresInstance) InspectManagedRoles(config config.Config, pgconn *pgx.Conn) error {
	if nil == config.Postgres.ManagedRolesQuery.Value {
		instance.ManagedRoles = instance.AllRoles
	} else {
		instance.ManagedRoles = make(roles.RoleSet)
		names, err := postgres.RunQuery(config.Postgres.ManagedRolesQuery, pgconn, postgres.RowToString, postgres.YamlToString)
		if err != nil {
			return err
		}
		for _, name := range names {
			match := instance.RolesBlacklist.MatchString(name)
			if "" == match {
				instance.ManagedRoles[name] = instance.AllRoles[name]
				slog.Debug("Managing Postgres role.", "name", name)

			} else {
				slog.Warn("Managed role is blacklisted.", "name", name, "pattern", match)
			}

		}
	}
	return nil
}
