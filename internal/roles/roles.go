package roles

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

type Role struct {
	Name    string
	Comment string
	Parents mapset.Set[string]
	Options config.RoleOptions
}

type RoleSet map[string]Role

func (rs RoleSet) Flatten() []string {
	var names []string
	seen := mapset.NewSet[string]()
	for _, role := range rs {
		for name := range rs.flattenRole(role, &seen) {
			names = append(names, name)
		}
	}
	return names
}

func (rs RoleSet) flattenRole(r Role, seen *mapset.Set[string]) (ch chan string) {
	ch = make(chan string)
	go func() {
		defer close(ch)
		if (*seen).Contains(r.Name) {
			return
		}
		for parentName := range r.Parents.Iter() {
			parent, ok := rs[parentName]
			if !ok {
				slog.Debug("Role herits from unknown parent.", "role", r.Name, "parent", parentName)
				continue
			}
			for deepName := range rs.flattenRole(parent, seen) {
				ch <- deepName
			}
		}

		(*seen).Add(r.Name)
		ch <- r.Name
	}()
	return
}

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

	optionsString := r.Options.String()
	wantedOptionsString := wanted.Options.String()
	if wantedOptionsString != optionsString {
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

	missingParents := wanted.Parents.Difference(r.Parents)
	if missingParents.Cardinality() > 0 {
		var parentIdentifiers []interface{}
		for parent := range missingParents.Iter() {
			parentIdentifiers = append(parentIdentifiers, pgx.Identifier{parent})
		}
		ch <- postgres.SyncQuery{
			Description: "Grant missing parents.",
			LogArgs: []interface{}{
				"role", r.Name,
				"parents", missingParents,
			},
			Query:     `GRANT %s TO %s;`,
			QueryArgs: []interface{}{parentIdentifiers, identifier},
		}
	}
	spuriousParents := r.Parents.Difference(wanted.Parents)
	if spuriousParents.Cardinality() > 0 {
		var parentIdentifiers []interface{}
		for parent := range spuriousParents.Iter() {
			parentIdentifiers = append(parentIdentifiers, pgx.Identifier{parent})
		}
		ch <- postgres.SyncQuery{
			Description: "Revoke spurious parents.",
			LogArgs: []interface{}{
				"role", r.Name,
				"parents", spuriousParents,
			},
			Query:     `REVOKE %s FROM %s;`,
			QueryArgs: []interface{}{parentIdentifiers, identifier},
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

	if 0 < r.Parents.Cardinality() {
		parents := []interface{}{}
		for parent := range r.Parents.Iter() {
			parents = append(parents, pgx.Identifier{parent})
		}
		ch <- postgres.SyncQuery{
			Description: "Create role.",
			LogArgs:     []interface{}{"role", r.Name, "parents", r.Parents},
			Query: `
			CREATE ROLE %s
			WITH ` + r.Options.String() + `
			IN ROLE %s;`,
			QueryArgs: []interface{}{identifier, parents},
		}
	} else {
		ch <- postgres.SyncQuery{
			Description: "Create role.",
			LogArgs:     []interface{}{"role", r.Name},
			Query:       `CREATE ROLE %s WITH ` + r.Options.String() + `;`,
			QueryArgs:   []interface{}{identifier},
		}
	}
	ch <- postgres.SyncQuery{
		Description: "Set role comment.",
		LogArgs:     []interface{}{"role", r.Name},
		Query:       `COMMENT ON ROLE %s IS %s;`,
		QueryArgs:   []interface{}{identifier, r.Comment},
	}
}

func (r *Role) Drop(databases []postgres.Database, currentUser Role, fallbackOwner string, ch chan postgres.SyncQuery) {
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
	if !currentUser.Options.Super {
		// Non-super user needs to inherit to-be-dropped role to reassign objects.
		if r.Parents.Contains(currentUser.Name) {
			// First, avoid membership loop.
			ch <- postgres.SyncQuery{
				Description: "Revoke membership on current user.",
				LogArgs: []interface{}{
					"role", r.Name, "parent", currentUser.Name,
				},
				Query: `REVOKE %s FROM %s;`,
				QueryArgs: []interface{}{
					pgx.Identifier{currentUser.Name},
					identifier,
				},
			}
		}
		ch <- postgres.SyncQuery{
			Description: "Allow current user to reassign objects.",
			LogArgs: []interface{}{
				"role", r.Name, "parent", currentUser.Name,
			},
			Query: `GRANT %s TO %s;`,
			QueryArgs: []interface{}{
				identifier,
				pgx.Identifier{currentUser.Name},
			},
		}
	}
	for i, database := range databases {
		if database.Owner == r.Name {
			ch <- postgres.SyncQuery{
				Description: "Reassign database.",
				LogArgs: []interface{}{
					"role", r.Name,
					"db", database.Name,
					"owner", fallbackOwner,
				},
				Query: `ALTER DATABASE %s OWNER TO %s;`,
				QueryArgs: []interface{}{
					pgx.Identifier{database.Name},
					pgx.Identifier{fallbackOwner},
				},
			}
			// Update model to generate propery queries next.
			databases[i].Owner = fallbackOwner
		}
		ch <- postgres.SyncQuery{
			Description: "Reassign objects and purge ACL.",
			LogArgs: []interface{}{
				"role", r.Name, "db", database.Name, "owner", database.Owner,
			},
			Database: database.Name,
			Query: `
			REASSIGN OWNED BY %s TO %s;
			DROP OWNED BY %s;`,
			QueryArgs: []interface{}{
				identifier, pgx.Identifier{database.Owner}, identifier,
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
