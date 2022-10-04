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
	roles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (role Role, err error) {
		role, err = NewRoleFromRow(row, instance.RoleColumns)
		return
	})
	if err != nil {
		log.Error("Failed to fetch rows.")
		return
	}

	for _, role := range roles {
		log.
			WithField("name", role.Name).
			WithField("super", role.Super).
			Debug("Found role in Postgres instance.")
	}

	instance.AllRoles = roles
	return
}
