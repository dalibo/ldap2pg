package inspect

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type PrivilegeInspecter interface {
	Databases(m postgres.DBMap, defaultDatabase string) []string
	Inspect() string
	RowTo(pgx.CollectableRow) (privilege.Grant, error)
}

var privilegeMap map[string]PrivilegeInspecter

//go:embed sql/schemas.sql
var schemasQuery string

func init() {
	// Glue code to instantiate concrete inspector from generic
	// implementation.
	privilegeMap = make(map[string]PrivilegeInspecter)
	for k, p := range privilege.Builtins {
		var i PrivilegeInspecter

		if "GLOBAL DEFAULT" == p.Object {
			i = privilege.NewGlobalDefault(p.Object, p.Inspect)
		} else if "SCHEMA DEFAULT" == p.Object {
			i = privilege.NewSchemaDefault(p.Object, p.Inspect)
		} else if strings.HasPrefix(p.Object, "ALL ") {
			i = privilege.NewAll(p.Object, p.Inspect)
		} else if "instance" == p.Scope {
			i = privilege.NewInstance(p.Object, p.Inspect)
		} else if "database" == p.Scope {
			i = privilege.NewDatabase(p.Object, p.Inspect)
		} else {
			continue
		}
		privilegeMap[k] = i
	}
}

func (instance *Instance) InspectStage2(ctx context.Context, pc Config) error {
	err := instance.InspectSchemas(ctx, pc.SchemasQuery)
	if err != nil {
		return fmt.Errorf("schemas: %w", err)
	}

	err = instance.InspectGrants(ctx, pc.ManagedPrivileges)
	if err != nil {
		return fmt.Errorf("privileges: %w", err)
	}

	return nil
}

func (instance *Instance) InspectGrants(ctx context.Context, managedPrivileges map[string][]string) error {
	slog.Info("Inspecting privileges.")
	for object, p := range privilegeMap {
		arg, ok := managedPrivileges[object]
		if !ok {
			continue
		}

		for _, database := range p.Databases(instance.Databases, instance.DefaultDatabase) {
			slog.Debug("Inspecting grants.", "database", database, "object", p)
			pgconn, err := postgres.DBPool.Get(ctx, database)
			if err != nil {
				return err
			}

			sql := p.Inspect()
			slog.Debug("Executing SQL query:\n"+sql, "arg", arg)
			rows, err := pgconn.Query(ctx, sql, arg)
			if err != nil {
				return fmt.Errorf("bad query: %w", err)
			}
			for rows.Next() {
				grant, err := p.RowTo(rows)
				if err != nil {
					return fmt.Errorf("bad row: %w", err)
				}
				grant.Database = database

				database, known := instance.Databases[grant.Database]
				if !known {
					continue
				}

				if "" != grant.Schema {
					_, known = database.Schemas[grant.Schema]
					if !known {
						continue
					}
				}

				pattern := instance.RolesBlacklist.MatchString(grant.Grantee)
				if pattern != "" {
					continue
				}

				pattern = instance.RolesBlacklist.MatchString(grant.Owner)
				if pattern != "" {
					continue
				}

				grant.Normalize()

				slog.Debug("Found grant in Postgres instance.", "grant", grant)
				instance.Grants = append(instance.Grants, grant)
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}

		}
	}
	return nil
}

func (instance *Instance) InspectSchemas(ctx context.Context, managedQuery Querier[postgres.Schema]) error {
	sq := &SQLQuery[postgres.Schema]{SQL: schemasQuery, RowTo: postgres.RowToSchema}

	for i, database := range instance.Databases {
		var managedSchemas []string
		slog.Debug("Inspecting managed schemas.", "database", database.Name)
		conn, err := postgres.DBPool.Get(ctx, database.Name)
		if err != nil {
			return err
		}
		for managedQuery.Query(ctx, conn); managedQuery.Next(); {
			s := managedQuery.Row()
			managedSchemas = append(managedSchemas, s.Name)
		}
		err = managedQuery.Err()
		if err != nil {
			return err
		}

		for sq.Query(ctx, conn); sq.Next(); {
			s := sq.Row()
			if !slices.Contains(managedSchemas, s.Name) {
				continue
			}
			database.Schemas[s.Name] = s
			slog.Debug("Found schema.", "db", database.Name, "schema", s.Name, "owner", s.Owner)
		}
		err = sq.Err()
		if err != nil {
			return err
		}

		instance.Databases[i] = database
	}

	return nil
}
