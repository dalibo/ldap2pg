package inspect

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

//go:embed sql/schemas.sql
var schemasQuery string

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
	for _, p := range privilege.Map {
		arg, ok := managedPrivileges[p.Object]
		if !ok {
			slog.Debug("Skipping privilege.", "object", p.Object)
			continue
		}

		var databases []string
		if "instance" == p.Scope {
			databases = []string{instance.DefaultDatabase}
		} else {
			databases = maps.Keys(instance.Databases)
		}

		for _, database := range databases {
			if p.IsDefault() {
				slog.Debug("Inspecting default grants.", "database", database, "scope", p.Object)
			} else {
				slog.Debug("Inspecting grants.", "scope", p.Scope, "database", database, "object", p.Object)
			}
			pgconn, err := postgres.DBPool.Get(ctx, database)
			if err != nil {
				return err
			}

			slog.Debug("Executing SQL query:\n"+p.Inspect, "arg", arg)
			rows, err := pgconn.Query(ctx, p.Inspect, arg)
			if err != nil {
				return fmt.Errorf("bad query: %w", err)
			}
			for rows.Next() {
				grant, err := privilege.RowTo(rows)
				if err != nil {
					return fmt.Errorf("bad row: %w", err)
				}
				if p.IsDefault() {
					grant.Target = grant.Object
				} else {
					grant.Target = p.Object
				}

				database, known := instance.Databases[grant.Database]
				if !known {
					slog.Debug("Ignoring grant on unmanaged database.", "database", grant.Database)
					continue
				}

				if "" != grant.Schema {
					_, known = database.Schemas[grant.Schema]
					if !known {
						slog.Debug("Ignoring grant on unmanaged schema.", "database", grant.Database, "schema", grant.Schema)
						continue
					}
				}

				pattern := instance.RolesBlacklist.MatchString(grant.Grantee)
				if pattern != "" {
					slog.Debug(
						"Ignoring grant to blacklisted role.",
						"grant", grant, "pattern", pattern)
					continue
				}

				pattern = instance.RolesBlacklist.MatchString(grant.Owner)
				if pattern != "" {
					slog.Debug(
						"Ignoring default grant for blacklisted role.",
						"grant", grant, "pattern", pattern)
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
