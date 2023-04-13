package config

import (
	"golang.org/x/exp/slog"
)

// Either an SQL string or a predefined list of YAML rows.
type SQLOrRows interface{}

// Like pgx.RowToFunc, but from YAML
type YamlToFunc[T any] func(row interface{}) (T, error)

type InspectQuery struct {
	Name    string
	Default SQLOrRows
	Value   SQLOrRows
}

func (q *InspectQuery) IsPredefined() bool {
	switch q.Value.(type) {
	case string:
		return false
	default:
		return true
	}
}

// Maybe set value from default.
func (q *InspectQuery) SetDefault() {
	if nil == q.Value {
		slog.Debug("Loading Postgres query from default.", "query", q)
		q.Value = q.Default
	}
}

func (q *InspectQuery) String() string {
	return q.Name
}
