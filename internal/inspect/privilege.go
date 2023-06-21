package inspect

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

func (instance *Instance) InspectStage2(ctx context.Context, pc Config) (err error) {
	err = instance.InspectGrants(ctx, pc.ManagedPrivileges)
	return
}

func (instance *Instance) InspectGrants(ctx context.Context, managedPrivileges map[string][]string) error {
	for _, p := range privilege.Map {
		managedTypes := managedPrivileges[p.Object]
		if 0 == len(managedTypes) {
			continue
		}
		var databases []string
		if "instance" == p.Scope {
			databases = []string{instance.DefaultDatabase}
		} else {
			databases = maps.Keys(instance.Databases)
		}

		for _, database := range databases {
			slog.Debug("Inspecting grants.", "scope", p.Scope, "database", database, "object", p.Object, "types", managedTypes)
			pgconn, err := postgres.DBPool.Get(ctx, database)
			if err != nil {
				return err
			}

			slog.Debug("Executing SQL query:\n"+p.Inspect, "arg", managedTypes)
			rows, err := pgconn.Query(ctx, p.Inspect, managedTypes)
			if err != nil {
				return fmt.Errorf("bad query: %w", err)
			}
			for rows.Next() {
				grant, err := privilege.RowTo(rows)
				if err != nil {
					return fmt.Errorf("bad row: %w", err)
				}
				grant.Target = p.Object

				database, known := instance.Databases[grant.Database]
				if !known {
					continue
				}
				if "" != grant.Schema && !slices.ContainsFunc(database.Schemas, func(s postgres.Schema) bool {
					return s.Name == grant.Schema
				}) {
					continue
				}

				pattern := instance.RolesBlacklist.MatchString(grant.Grantee)
				if pattern != "" {
					slog.Debug(
						"Ignoring grant to blacklisted role.",
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

func (instance *Instance) InspectSchemas(ctx context.Context, query Querier[postgres.Schema]) error {
	for i, database := range instance.Databases {
		conn, err := postgres.DBPool.Get(ctx, database.Name)
		if err != nil {
			return err
		}
		for query.Query(ctx, conn); query.Next(); {
			s := query.Row()
			database.Schemas = append(database.Schemas, s)
			slog.Debug("Found schema.", "db", database.Name, "schema", s.Name, "owner", s.Owner)
		}
		instance.Databases[i] = database
		err = query.Err()
		if err != nil {
			return fmt.Errorf("schemas: %w", err)
		}
	}

	return nil
}
