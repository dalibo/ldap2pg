package inspect

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

//go:embed sql/creators.sql
var creatorsQuery string

func (instance *Instance) InspectStage3(ctx context.Context, roles []string) error {
	err := instance.InspectCreators(ctx, roles)
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

func (instance *Instance) InspectCreators(ctx context.Context, managedRoles []string) error {
	slog.Info("Inspecting object creators.")
	cq := &SQLQuery[Creators]{SQL: creatorsQuery, RowTo: RowToCreators}

	for i, database := range instance.Databases {
		slog.Debug("Inspecting schemas creators.", "database", database.Name)
		conn, err := postgres.DBPool.Get(ctx, database.Name)
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
				if !slices.Contains(managedRoles, name) {
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

		instance.Databases[i] = database
	}

	return nil
}
