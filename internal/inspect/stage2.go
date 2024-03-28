package inspect

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slices"
)

//go:embed sql/schemas.sql
var schemasQuery string

func (instance *Instance) InspectStage2(ctx context.Context, dbname string, query Querier[postgres.Schema]) error {
	err := instance.InspectSchemas(ctx, dbname, query)
	if err != nil {
		return fmt.Errorf("schemas: %w", err)
	}
	return nil
}

func (instance *Instance) InspectGrants(ctx context.Context, dbname string, privileges privilege.TypeMap, roles mapset.Set[string]) (out []privilege.Grant, err error) {
	inspector := privilege.NewInspector(instance.Databases[dbname], instance.DefaultDatabase, privileges)
	for inspector.Run(ctx); inspector.Next(); {
		grant := inspector.Grant()
		if grant.IsRelevant() && !roles.Contains(grant.Grantee) {
			continue
		}
		if grant.IsDefault() && !roles.Contains(grant.Owner) {
			continue
		}

		grant.Normalize()

		slog.Debug("Found grant in Postgres instance.", "grant", grant)
		out = append(out, grant)
	}
	err = inspector.Err()
	return
}

func (instance *Instance) InspectSchemas(ctx context.Context, dbname string, managedQuery Querier[postgres.Schema]) error {
	conn, err := postgres.GetConn(ctx, dbname)
	if err != nil {
		return err
	}

	sq := &SQLQuery[postgres.Schema]{SQL: schemasQuery, RowTo: postgres.RowToSchema}
	var managedSchemas []string
	slog.Debug("Inspecting managed schemas.", "database", dbname)
	for managedQuery.Query(ctx, conn); managedQuery.Next(); {
		s := managedQuery.Row()
		managedSchemas = append(managedSchemas, s.Name)
	}
	err = managedQuery.Err()
	if err != nil {
		return err
	}

	database := instance.Databases[dbname]
	for sq.Query(ctx, conn); sq.Next(); {
		s := sq.Row()
		if !slices.Contains(managedSchemas, s.Name) {
			continue
		}
		database.Schemas[s.Name] = s
		slog.Debug("Found schema.", "database", dbname, "schema", s.Name, "owner", s.Owner)
	}
	err = sq.Err()
	if err != nil {
		return err
	}

	instance.Databases[dbname] = database

	return nil
}
