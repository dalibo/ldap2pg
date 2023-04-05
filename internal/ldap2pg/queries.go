// Configurable and overridable queries.
package ldap2pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Either an SQL string or a predefined list of YAML rows.
type SQLOrRows interface{}

type Query struct {
	Name    string
	Default SQLOrRows
	Value   SQLOrRows
}

// Like pgx.RowToFunc, but from YAML
type YamlToFunc[T any] func(row interface{}) (T, error)

func RunQuery[T any](q Query, pgconn *pgx.Conn, pgFun pgx.RowToFunc[T], yamlFun YamlToFunc[T]) ([]T, error) {
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
	if err != nil {
		slog.Error("Bad query.", "error", err)
		return nil, err
	}
	return pgx.CollectRows(rows, pgFun)
}

func (q *Query) IsPredefined() bool {
	switch q.Value.(type) {
	case string:
		return false
	default:
		return true
	}
}

// Maybe set value from default.
func (q *Query) SetDefault() {
	if nil == q.Value {
		slog.Debug("Loading Postgres query from default.", "query", q)
		q.Value = q.Default
	}
}

func (q *Query) String() string {
	return q.Name
}

func RowToString(row pgx.CollectableRow) (pattern string, err error) {
	err = row.Scan(&pattern)
	return
}

func YamlToString(value interface{}) (pattern string, err error) {
	pattern = value.(string)
	return
}
