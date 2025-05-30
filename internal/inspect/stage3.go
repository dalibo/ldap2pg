package inspect

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/v6/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

//go:embed sql/creators.sql
var creatorsQuery string

func (instance *Instance) InspectStage3(ctx context.Context, dbname string, roles mapset.Set[string]) error {
	err := instance.InspectCreators(ctx, dbname, roles)
	if err != nil {
		return fmt.Errorf("creators: %w", err)
	}

	return nil
}

type Creators struct {
	Schema   string
	Creators []string
}

func RowToCreators(rows pgx.CollectableRow) (c Creators, err error) {
	err = rows.Scan(&c.Schema, &c.Creators)
	return
}

func (instance *Instance) InspectCreators(ctx context.Context, dbname string, managedRoles mapset.Set[string]) error {
	cq := &SQLQuery[Creators]{SQL: creatorsQuery, RowTo: RowToCreators}

	database := postgres.Databases[dbname]
	slog.Debug("Inspecting objects creators.", "database", dbname)
	conn, err := postgres.GetConn(ctx, dbname)
	if err != nil {
		return err
	}

	for cq.Query(ctx, conn); cq.Next(); {
		c := cq.Row()
		s, ok := database.Schemas[c.Schema]
		if !ok {
			continue
		}

		for _, name := range c.Creators {
			if !managedRoles.Contains(name) {
				continue
			}
			s.Creators = append(s.Creators, name)
		}
		slog.Debug("Found schema creators.", "database", database.Name, "schema", s.Name, "owner", s.Owner, "creators", s.Creators)
		database.Schemas[c.Schema] = s
	}
	err = cq.Err()
	if err != nil {
		return err
	}

	postgres.Databases[dbname] = database

	return nil
}
