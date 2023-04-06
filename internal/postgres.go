package internal

import (
	"context"
	_ "embed"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type PostgresInstance struct {
	AllRoles       RoleSet
	ManagedRoles   RoleSet
	RoleColumns    []string
	RolesBlacklist Blacklist
}

var (
	//go:embed sql/role-columns.sql
	roleColumnsQuery string
	//go:embed sql/roles.sql
	rolesQuery string
)

func PostgresInspect(config Config) (instance PostgresInstance, err error) {
	instance = PostgresInstance{}

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	patterns, err := RunQuery(config.Postgres.RolesBlacklistQuery, pgconn, RowToString, YamlToString)
	if err != nil {
		return
	}
	instance.RolesBlacklist = Blacklist(patterns)

	rows, err := pgconn.Query(ctx, roleColumnsQuery)
	if err != nil {
		slog.Error("Failed to query role columns.")
		return
	}
	columns, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		slog.Error("Failed to fetch rows.")
		return
	}
	instance.RoleColumns = columns
	slog.Debug("Querying PostgreSQL instance role columns.", "columns", instance.RoleColumns)

	sql := "rol." + strings.Join(instance.RoleColumns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	rows, err = pgconn.Query(ctx, rolesQuery)
	if err != nil {
		slog.Error("Failed to query role columns.")
		return
	}
	unfilteredRoles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (role Role, err error) {
		role, err = NewRoleFromRow(row, instance.RoleColumns)
		return
	})
	if err != nil {
		slog.Error("Failed to fetch rows.")
		return
	}

	instance.AllRoles = make(RoleSet)
	for _, role := range unfilteredRoles {
		match := instance.RolesBlacklist.Match(&role)
		if match == "" {
			instance.AllRoles[role.Name] = role
			slog.Debug("Found role in Postgres instance.", "name", role.Name, "super", role.Options.Super)

		} else {
			slog.Debug("Role name blacklisted. Ignoring.", "name", role.Name, "pattern", match)
		}
	}

	err = instance.InspectManagedRoles(config, pgconn)
	return
}

func (instance *PostgresInstance) InspectManagedRoles(config Config, pgconn *pgx.Conn) error {
	if nil == config.Postgres.ManagedRolesQuery.Value {
		instance.ManagedRoles = instance.AllRoles
	} else {
		instance.ManagedRoles = make(RoleSet)
		names, err := RunQuery(config.Postgres.ManagedRolesQuery, pgconn, RowToString, YamlToString)
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
