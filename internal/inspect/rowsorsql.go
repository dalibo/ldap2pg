package inspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Either an SQL string or a predefined list of YAML rows.
type RowsOrSQL struct {
	Value interface{}
}

// Like pgx.RowToFunc, but from YAML
type YamlToFunc[T any] func(row interface{}) (T, error)

func IsPredefined(q RowsOrSQL) bool {
	switch q.Value.(type) {
	case string:
		return false
	default:
		return true
	}
}

// Implements inspect.YamlToFunc. Similar to pgx.RowTo.
func YamlToString(value interface{}) (pattern string, err error) {
	pattern = value.(string)
	return
}

func RunQuery[T any](q interface{}, pgconn *pgx.Conn, pgFun pgx.RowToFunc[T], yamlFun YamlToFunc[T]) <-chan any {
	ch := make(chan any)
	go func() {
		defer close(ch)
		var sql string
		rowsOrSQL, ok := q.(RowsOrSQL)
		if ok {
			if IsPredefined(rowsOrSQL) {
				slog.Debug("Reading values from YAML.")
				for _, value := range rowsOrSQL.Value.([]interface{}) {
					row, err := yamlFun(value)
					if err != nil {
						ch <- err
					} else {
						ch <- row
					}
				}
				return
			}
			sql = rowsOrSQL.Value.(string)
		} else {
			sql = q.(string)
		}

		ctx := context.Background()
		rows, err := pgconn.Query(ctx, sql)
		slog.Debug("Executing SQL query:\n" + sql)
		if err != nil {
			ch <- fmt.Errorf("Bad query: %w", err)
		}
		for rows.Next() {
			rowData, err := pgFun(rows)
			if err != nil {
				ch <- err
			} else {
				ch <- rowData
			}
		}
	}()
	return ch
}
