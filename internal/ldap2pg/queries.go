// Configurable and overridable queries.
package ldap2pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
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
		log.WithField("error", err).Error("Bad query.")
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
		log.
			WithField("query", q).
			Debug("Loading Postgres query from default.")
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
