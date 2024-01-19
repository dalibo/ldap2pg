package role

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

type Role struct {
	Name       string
	Comment    string
	Parents    mapset.Set[string]
	Options    Options
	Config     *Config
	Manageable bool
}

func New() Role {
	r := Role{}
	r.Parents = mapset.NewSet[string]()
	r.Config = &Config{}
	return r
}

func RowTo(row pgx.CollectableRow) (r Role, err error) {
	var variableRow interface{}
	var parents []string
	var config []string
	r = New()
	err = row.Scan(&r.Name, &variableRow, &r.Comment, &parents, &config, &r.Manageable)
	if err != nil {
		return
	}
	r.Parents.Append(parents...)
	r.Options.LoadRow(variableRow.([]interface{}))
	(*r.Config).Parse(config)
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
func (r *Role) Alter(wanted Role, serverVersionNum int) (out []postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}

	// It's so evident that wanted role has to be manageable. don't even
	// compare with wanted state.
	if !r.Manageable && 160000 > serverVersionNum {
		out = append(out, postgres.SyncQuery{
			Description: "Inherit role for management.",
			LogArgs:     []interface{}{"role", r.Name},
			Query:       `GRANT %s TO CURRENT_USER WITH ADMIN OPTION;`,
			QueryArgs:   []interface{}{identifier},
		})
	}

	wantedOptions := wanted.Options.Diff(r.Options)
	if wantedOptions != "" {
		out = append(out, postgres.SyncQuery{
			Description: "Alter options.",
			LogArgs: []interface{}{
				"role", r.Name,
				"options", wantedOptions,
			},
			Query:     `ALTER ROLE %s WITH ` + wantedOptions + `;`,
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

	if wanted.Config != nil {
		currentKeys := mapset.NewSetFromMapKeys(*r.Config)
		wantedKeys := mapset.NewSetFromMapKeys(*wanted.Config)
		missingKeys := wantedKeys.Clone()
		for k := range currentKeys.Iter() {
			if !wantedKeys.Contains(k) {
				out = append(out, postgres.SyncQuery{
					Description: "Reset role config.",
					LogArgs: []interface{}{
						"role", r.Name,
						"config", k,
					},
					Query:     `ALTER ROLE %s RESET %s;`,
					QueryArgs: []interface{}{identifier, pgx.Identifier{k}},
				})
				continue
			}

			missingKeys.Remove(k)

			currentValue := (*r.Config)[k]
			wantedValue := (*wanted.Config)[k]
			if wantedValue == currentValue {
				continue
			}
			out = append(out, postgres.SyncQuery{
				Description: "Update role config.",
				LogArgs: []interface{}{
					"role", r.Name,
					"config", k,
					"current", currentValue,
					"wanted", wantedValue,
				},
				Query:     `ALTER ROLE %s SET %s TO %s;`,
				QueryArgs: []interface{}{identifier, pgx.Identifier{k}, wantedValue},
			})
		}

		for k := range missingKeys.Iter() {
			v := (*wanted.Config)[k]
			out = append(out, postgres.SyncQuery{
				Description: "Set role config.",
				LogArgs: []interface{}{
					"role", r.Name,
					"config", k,
					"value", v,
				},
				Query:     `ALTER ROLE %s SET %s TO %s;`,
				QueryArgs: []interface{}{identifier, pgx.Identifier{k}, v},
			})
		}
	}

	return
}

func (r *Role) Create(super bool, serverVersionNum int) (out []postgres.SyncQuery) {
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

	if !super {
		if 160000 > serverVersionNum {
			out = append(out, postgres.SyncQuery{
				Description: "Inherit role for management.",
				LogArgs:     []interface{}{"role", r.Name},
				Query:       `GRANT %s TO CURRENT_USER WITH ADMIN OPTION;`,
				QueryArgs:   []interface{}{identifier},
			})
		} else {
			if r.Parents.Contains("owners") {
				out = append(out, postgres.SyncQuery{
					Description: "Inherit role for management.",
					LogArgs:     []interface{}{"role", r.Name},
					Query:       `GRANT %s TO CURRENT_USER WITH INHERIT OPTION;`,
					QueryArgs:   []interface{}{identifier},
				})
			}
		}
	}

	if nil == r.Config {
		return
	}

	for k, v := range *r.Config {
		out = append(out, postgres.SyncQuery{
			Description: "Set role config.",
			LogArgs:     []interface{}{"role", r.Name, "config", k, "value", v},
			Query:       `ALTER ROLE %s SET %s TO %s`,
			QueryArgs:   []interface{}{identifier, pgx.Identifier{k}, v},
		})
	}
	return
}

func (r *Role) Drop(databases *postgres.DBMap, currentUser Role, fallbackOwner string) (out []postgres.SyncQuery) {
	identifier := pgx.Identifier{r.Name}
	if r.Options.CanLogin {
		out = append(out, postgres.SyncQuery{
			Description: "Terminate running sessions.",
			LogArgs:     []interface{}{"role", r.Name},
			Database:    "<first>",
			Query: `
			SELECT pg_terminate_backend(pid)
			FROM pg_catalog.pg_stat_activity
			WHERE usename = %s;`,
			QueryArgs: []interface{}{r.Name},
		})
	}

	if !currentUser.Options.Super {
		// Non-super user needs to inherit to-be-dropped role to reassign objects.
		if r.Parents.Contains(currentUser.Name) {
			// First, avoid membership loop.
			out = append(out, postgres.SyncQuery{
				Description: "Revoke membership on current user.",
				LogArgs: []interface{}{
					"role", r.Name, "parent", currentUser.Name,
				},
				Database: "<first>",
				Query:    `REVOKE %s FROM %s;`,
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
			Database: "<first>",
			Query:    `GRANT %s TO %s;`,
			QueryArgs: []interface{}{
				identifier,
				pgx.Identifier{currentUser.Name},
			},
		})
	}
	for dbname, database := range *databases {
		if database.Owner == r.Name {
			out = append(out, postgres.SyncQuery{
				Description: "Reassign database.",
				LogArgs: []interface{}{
					"database", database.Name,
					"old", r.Name,
					"new", fallbackOwner,
				},
				Query: `ALTER DATABASE %s OWNER TO %s;`,
				QueryArgs: []interface{}{
					pgx.Identifier{database.Name},
					pgx.Identifier{fallbackOwner},
				},
			})
			// Update model to generate propery queries next.
			database.Owner = fallbackOwner
			(*databases)[dbname] = database
		}
		out = append(out, postgres.SyncQuery{
			Description: "Reassign objects and purge ACL.",
			LogArgs: []interface{}{
				"role", r.Name, "owner", database.Owner,
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

func (r *Role) Merge(o Role) {
	r.Parents.Append(o.Parents.ToSlice()...)
}
