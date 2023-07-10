package role

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

type Role struct {
	Name    string
	Comment string
	Parents mapset.Set[string]
	Options Options
}

func New() Role {
	role := Role{}
	role.Parents = mapset.NewSet[string]()
	return role
}

func RowTo(row pgx.CollectableRow) (role Role, err error) {
	var variableRow interface{}
	var parents []string
	role = New()
	err = row.Scan(&role.Name, &variableRow, &role.Comment, &parents)
	if err != nil {
		return
	}
	role.Parents.Append(parents...)
	role.Options.LoadRow(variableRow.([]interface{}))
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
func (r *Role) Alter(wanted Role) (out []postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}

	optionsString := r.Options.String()
	wantedOptionsString := wanted.Options.String()
	if wantedOptionsString != optionsString {
		out = append(out, postgres.SyncQuery{
			Description: "Alter options.",
			LogArgs: []interface{}{
				"role", r.Name,
				"current", r.Options,
				"wanted", wanted.Options,
			},
			Query:     `ALTER ROLE %s WITH ` + wanted.Options.String() + `;`,
			QueryArgs: []interface{}{identifier},
		})
	}

	missingParents := wanted.Parents.Difference(r.Parents)
	if missingParents.Cardinality() > 0 {
		var parentIdentifiers []interface{}
		for parent := range missingParents.Iter() {
			parentIdentifiers = append(parentIdentifiers, pgx.Identifier{parent})
		}
		out = append(out, postgres.SyncQuery{
			Description: "Grant missing parents.",
			LogArgs: []interface{}{
				"role", r.Name,
				"parents", missingParents,
			},
			Query:     `GRANT %s TO %s;`,
			QueryArgs: []interface{}{parentIdentifiers, identifier},
		})
	}
	spuriousParents := r.Parents.Difference(wanted.Parents)
	if spuriousParents.Cardinality() > 0 {
		var parentIdentifiers []interface{}
		for parent := range spuriousParents.Iter() {
			parentIdentifiers = append(parentIdentifiers, pgx.Identifier{parent})
		}
		out = append(out, postgres.SyncQuery{
			Description: "Revoke spurious parents.",
			LogArgs: []interface{}{
				"role", r.Name,
				"parents", spuriousParents,
			},
			Query:     `REVOKE %s FROM %s;`,
			QueryArgs: []interface{}{parentIdentifiers, identifier},
		})
	}

	if wanted.Comment != r.Comment {
		out = append(out, postgres.SyncQuery{
			Description: "Set role comment.",
			LogArgs: []interface{}{
				"role", r.Name,
				"current", r.Comment,
				"wanted", wanted.Comment,
			},
			Query:     `COMMENT ON ROLE %s IS %s;`,
			QueryArgs: []interface{}{identifier, wanted.Comment},
		})
	}
	return
}

func (r *Role) Create() (out []postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}

	if 0 < r.Parents.Cardinality() {
		parents := []interface{}{}
		for parent := range r.Parents.Iter() {
			parents = append(parents, pgx.Identifier{parent})
		}
		out = append(out, postgres.SyncQuery{
			Description: "Create role.",
			LogArgs:     []interface{}{"role", r.Name, "parents", r.Parents.ToSlice()},
			Query: `
			CREATE ROLE %s
			WITH ` + r.Options.String() + `
			IN ROLE %s;`,
			QueryArgs: []interface{}{identifier, parents},
		})
	} else {
		out = append(out, postgres.SyncQuery{
			Description: "Create role.",
			LogArgs:     []interface{}{"role", r.Name},
			Query:       `CREATE ROLE %s WITH ` + r.Options.String() + `;`,
			QueryArgs:   []interface{}{identifier},
		})
	}
	out = append(out, postgres.SyncQuery{
		Description: "Set role comment.",
		LogArgs:     []interface{}{"role", r.Name},
		Query:       `COMMENT ON ROLE %s IS %s;`,
		QueryArgs:   []interface{}{identifier, r.Comment},
	})
	return
}

func (r *Role) Drop(databases postgres.DBMap, currentUser Role, fallbackOwner string) (out []postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}
	out = append(out, postgres.SyncQuery{
		Description: "Terminate running sessions.",
		LogArgs:     []interface{}{"role", r.Name},
		Query: `
		SELECT pg_terminate_backend(pid)
		FROM pg_catalog.pg_stat_activity
		WHERE usename = %s;`,
		QueryArgs: []interface{}{r.Name},
	})

	if !currentUser.Options.Super {
		// Non-super user needs to inherit to-be-dropped role to reassign objects.
		if r.Parents.Contains(currentUser.Name) {
			// First, avoid membership loop.
			out = append(out, postgres.SyncQuery{
				Description: "Revoke membership on current user.",
				LogArgs: []interface{}{
					"role", r.Name, "parent", currentUser.Name,
				},
				Query: `REVOKE %s FROM %s;`,
				QueryArgs: []interface{}{
					pgx.Identifier{currentUser.Name},
					identifier,
				},
			})
		}
		out = append(out, postgres.SyncQuery{
			Description: "Allow current user to reassign objects.",
			LogArgs: []interface{}{
				"role", r.Name, "parent", currentUser.Name,
			},
			Query: `GRANT %s TO %s;`,
			QueryArgs: []interface{}{
				identifier,
				pgx.Identifier{currentUser.Name},
			},
		})
	}
	for dbname, database := range databases {
		if database.Owner == r.Name {
			out = append(out, postgres.SyncQuery{
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
			})
			// Update model to generate propery queries next.
			database.Owner = fallbackOwner
			databases[dbname] = database
		}
		out = append(out, postgres.SyncQuery{
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
		})
	}
	out = append(out, postgres.SyncQuery{
		Description: "Drop role.",
		LogArgs:     []interface{}{"role", r.Name},
		Query:       `DROP ROLE %s;`,
		QueryArgs:   []interface{}{identifier},
	})
	return
}
