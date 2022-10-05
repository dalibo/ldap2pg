package ldap2pg

import (
	"context"
	_ "embed"
	"strings"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
)

// Fourzitou struct holding everything need to synchronize Instance.
type PostgresInstance struct {
	AllRoles       []Role
	ManagedRoles   []string
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
		log.Error("Failed to query role columns.")
		return
	}
	columns, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		log.Error("Failed to fetch rows.")
		return
	}
	instance.RoleColumns = columns
	log.WithField("columns", instance.RoleColumns).
		Debug("Querying PostgreSQL instance role columns.")

	sql := "rol." + strings.Join(instance.RoleColumns, ", rol.")
	rolesQuery = strings.Replace(rolesQuery, "rol.*", sql, 1)
	rows, err = pgconn.Query(ctx, rolesQuery)
	if err != nil {
		log.Error("Failed to query role columns.")
		return
	}
	unfilteredRoles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (role Role, err error) {
		role, err = NewRoleFromRow(row, instance.RoleColumns)
		return
	})
	if err != nil {
		log.Error("Failed to fetch rows.")
		return
	}

	var roles []Role
	for _, role := range unfilteredRoles {
		match := instance.RolesBlacklist.Match(&role)
		if match == "" {
			roles = append(roles, role)
			log.
				WithField("name", role.Name).
				WithField("super", role.Super).
				Debug("Found role in Postgres instance.")
		} else {
			log.
				WithField("name", role.Name).
				WithField("pattern", match).
				Debug("Role name blacklisted. Ignoring.")
		}
	}

	instance.AllRoles = roles
	err = instance.InspectManagedRoles(config, pgconn)
	return
}

func (instance *PostgresInstance) InspectManagedRoles(config Config, pgconn *pgx.Conn) error {
	if nil == config.Postgres.ManagedRolesQuery.Value {
		for _, role := range instance.AllRoles {
			instance.ManagedRoles = append(instance.ManagedRoles, role.Name)
		}
	} else {
		names, err := RunQuery(config.Postgres.ManagedRolesQuery, pgconn, RowToString, YamlToString)
		if err != nil {
			return err
		}
		for _, name := range names {
			match := instance.RolesBlacklist.MatchString(name)
			if "" == match {
				log.
					WithField("name", name).
					Debug("Managing Postgres role.")
				instance.ManagedRoles = append(instance.ManagedRoles, name)
			} else {
				log.
					WithField("name", name).
					WithField("pattern", match).
					Warning("Managed role is blacklisted.")
			}

		}
	}
	return nil
}
