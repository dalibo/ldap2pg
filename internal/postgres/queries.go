// Configurable and overridable queries.
package postgres

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

type SyncQuery struct {
	Description string
	LogArgs     []interface{}
	Database    string
	Query       string
	QueryArgs   []interface{}
}

func (q SyncQuery) String() string {
	return q.Description
}

func RunQuery[T any](q config.InspectQuery, pgconn *pgx.Conn, pgFun pgx.RowToFunc[T], yamlFun config.YamlToFunc[T]) ([]T, error) {
	if q.IsPredefined() {
		var rows []T
		for _, value := range q.Value.([]interface{}) {
			row, err := yamlFun(value)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
		return rows, nil
	}

	ctx := context.Background()
	rows, err := pgconn.Query(ctx, q.Value.(string))
	slog.Debug(q.Value.(string))
	if err != nil {
		err = fmt.Errorf("Bad query: %w", err)
		return nil, err
	}
	return pgx.CollectRows(rows, pgFun)
}

func RowToString(row pgx.CollectableRow) (pattern string, err error) {
	err = row.Scan(&pattern)
	return
}

func YamlToString(value interface{}) (pattern string, err error) {
	pattern = value.(string)
	return
}
