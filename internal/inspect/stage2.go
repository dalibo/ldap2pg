package inspect

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
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

func (instance *Instance) InspectSchemas(ctx context.Context, dbname string, managedQuery Querier[postgres.Schema]) error {
	conn, err := postgres.GetConn(ctx, dbname)
	if err != nil {
		return err
	}

	var managedSchemas []string
	slog.Debug("Inspecting managed schemas.", "config", "schemas_query", "database", dbname)
	for managedQuery.Query(ctx, conn); managedQuery.Next(); {
		s := managedQuery.Row()
		managedSchemas = append(managedSchemas, s.Name)
	}
	err = managedQuery.Err()
	if err != nil {
		return err
	}

	database := postgres.Databases[dbname]
	sq := &SQLQuery[postgres.Schema]{SQL: schemasQuery, RowTo: postgres.RowToSchema}
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

	postgres.Databases[dbname] = database

	return nil
}
