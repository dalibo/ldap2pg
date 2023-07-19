package inspect

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
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
	inspecter := privilege.NewInspector(instance.Databases, instance.DefaultDatabase, managedPrivileges)
	for inspecter.Run(ctx); inspecter.Next(); {
		grant := inspecter.Grant()
		pattern := instance.RolesBlacklist.MatchString(grant.Grantee)
		if pattern != "" {
			continue
		}

		pattern = instance.RolesBlacklist.MatchString(grant.Owner)
		if pattern != "" {
			continue
		}

		slog.Debug("Found grant in Postgres instance.", "grant", grant)
		instance.Grants = append(instance.Grants, grant)
	}

	return inspecter.Err()
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
