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

func PostgresInspect(c config.Config) (instance PostgresInstance, err error) {
	instance = PostgresInstance{}

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	err = instance.InspectDatabases(c, pgconn)
	if err != nil {
		return
	}

	err = instance.InspectRoles(c, pgconn)
	if err != nil {
		return
	}
	err = instance.InspectManagedRoles(c, pgconn)
	return
}

func (instance *PostgresInstance) InspectDatabases(c config.Config, pgconn *pgx.Conn) error {
	slog.Debug("Inspecting managed databases.")
	err := utils.IterateToSlice(postgres.RunQuery(c.Postgres.DatabasesQuery, pgconn, pgx.RowTo[string], config.YamlToString), &instance.Databases)
	if err != nil {
		return err
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
	err := utils.IterateToSlice(postgres.RunQuery(config.InspectQuery{Value: roleColumnsQuery}, pgconn, pgx.RowTo[string], nil), &instance.RoleColumns)
	if err != nil {
		return err
	}
	slog.Debug("Inspected PostgreSQL instance role options.", "columns", instance.RoleColumns)

	instance.AllRoles = make(roles.RoleSet)
	sql := "rol." + strings.Join(instance.RoleColumns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	slog.Debug("Inspecting all roles.")
	for item := range postgres.RunQuery(config.InspectQuery{Value: rolesQuery}, pgconn, func(row pgx.CollectableRow) (role roles.Role, err error) {
		role, err = roles.NewRoleFromRow(row, instance.RoleColumns)
		return
	}, nil) {
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
