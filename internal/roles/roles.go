package roles

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

type Role struct {
	Name    string
	Comment string
	Parents mapset.Set[string]
	Options config.RoleOptions
}

type RoleSet map[string]Role

func NewRoleFromRow(row pgx.CollectableRow, instanceRoleColumns []string) (role Role, err error) {
	var variableRow interface{}
	var parents []string
	err = row.Scan(&role.Name, &variableRow, &role.Comment, &parents)
	if err != nil {
		return
	}
	role.Parents = mapset.NewSet[string](parents...)
	record := variableRow.([]interface{})
	var colname string
	for i, value := range record {
		colname = instanceRoleColumns[i]
		switch colname {
		case "rolbypassrls":
			role.Options.ByPassRLS = value.(bool)
		case "rolcanlogin":
			role.Options.CanLogin = value.(bool)
		case "rolconnlimit":
			role.Options.ConnLimit = int(value.(int32))
		case "rolcreatedb":
			role.Options.CreateDB = value.(bool)
		case "rolcreaterole":
			role.Options.CreateRole = value.(bool)
		case "rolinherit":
			role.Options.Inherit = value.(bool)
		case "rolreplication":
			role.Options.Replication = value.(bool)
		case "rolsuper":
			role.Options.Super = value.(bool)
		}
	}
	return
}

func (r *Role) String() string {
	return r.Name
}

func (r *Role) BlacklistKey() string {
	return r.Name
}

// Generate queries to update current role configuration to match wanted role
// configuration.
func (r *Role) Alter(wanted Role, ch chan postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}

	if wanted.Options != r.Options {
		ch <- postgres.SyncQuery{
			Description: "Alter options.",
			LogArgs: []interface{}{
				"role", r.Name,
				"current", r.Options,
				"wanted", wanted.Options,
			},
			Query:     `ALTER ROLE %s WITH ` + wanted.Options.String() + `;`,
			QueryArgs: []interface{}{identifier},
		}
	}

	if wanted.Comment != r.Comment {
		ch <- postgres.SyncQuery{
			Description: "Set role comment.",
			LogArgs: []interface{}{
				"role", r.Name,
				"current", r.Comment,
				"wanted", wanted.Comment,
			},
			Query:     `COMMENT ON ROLE %s IS %s;`,
			QueryArgs: []interface{}{identifier, wanted.Comment},
		}
	}
}

func (r *Role) Create(ch chan postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}

	ch <- postgres.SyncQuery{
		Description: "Create role.",
		LogArgs: []interface{}{
			"role", r.Name,
		},
		Query:     `CREATE ROLE %s WITH ` + r.Options.String() + `;`,
		QueryArgs: []interface{}{identifier},
	}
	ch <- postgres.SyncQuery{
		Description: "Set role comment.",
		LogArgs: []interface{}{
			"role", r.Name,
		},
		Query:     `COMMENT ON ROLE %s IS %s;`,
		QueryArgs: []interface{}{identifier, r.Comment},
	}
}

func (r *Role) Drop(databases []string, ch chan postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}
	ch <- postgres.SyncQuery{
		Description: "Terminate running sessions.",
		LogArgs:     []interface{}{"role", r.Name},
		Query: `
		SELECT pg_terminate_backend(pid)
		FROM pg_catalog.pg_stat_activity
		WHERE usename = %s;`,
		QueryArgs: []interface{}{r.Name},
	}
	for _, database := range databases {
		ch <- postgres.SyncQuery{
			Description: "Reassign objects and purge ACL.",
			LogArgs:     []interface{}{"role", r.Name, "database", database},
			Database:    database,
			Query: `
			REASSIGN OWNED BY %s TO CURRENT_USER;
			DROP OWNED BY %s;`,
			QueryArgs: []interface{}{
				identifier, identifier,
			},
		}
	}
	ch <- postgres.SyncQuery{
		Description: "Drop role.",
		LogArgs:     []interface{}{"role", r.Name},
		Query:       `DROP ROLE %s;`,
		QueryArgs:   []interface{}{identifier},
	}
}
