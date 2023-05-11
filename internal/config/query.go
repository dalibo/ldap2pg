package config

import (
	"fmt"

	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/jackc/pgx/v5"
	"github.com/lithammer/dedent"
)

// PostgresConfig holds the configuration of an inspect.Config.
//
// This structure let mapstructure decode each query individually. The actually
// Querier object is instanciated early. Use Build() method to produce the
// final inspect.Config object.
type PostgresConfig struct {
	FallbackOwner       string              `mapstructure:"fallback_owner"`
	DatabasesQuery      QueryConfig[string] `mapstructure:"databases_query"`
	ManagedRolesQuery   QueryConfig[string] `mapstructure:"managed_roles_query"`
	RolesBlacklistQuery QueryConfig[string] `mapstructure:"roles_blacklist_query"`
	Privileges          privilege.RefMap    `mapstructure:"omit"`
}

func (c PostgresConfig) Build() inspect.Config {
	return inspect.Config{
		FallbackOwner:       c.FallbackOwner,
		DatabasesQuery:      c.DatabasesQuery.Querier,
		ManagedRolesQuery:   c.ManagedRolesQuery.Querier,
		RolesBlacklistQuery: c.RolesBlacklistQuery.Querier,
	}
}

type QueryConfig[T any] struct {
	Value   interface{}
	Querier inspect.Querier[T]
}

func NewSQLQuery[T any](sql string) QueryConfig[T] {
	return QueryConfig[T]{
		Querier: &inspect.SQLQuery[T]{
			SQL:   dedent.Dedent(sql),
			RowTo: pgx.RowTo[T],
		},
	}
}

func NewYAMLQuery[T any](rows ...T) QueryConfig[T] {
	return QueryConfig[T]{
		Querier: &inspect.YAMLQuery[T]{
			Rows: rows,
		},
	}
}

func (qc *QueryConfig[T]) Instantiate(rowTo pgx.RowToFunc[T], yamlTo YamlToFunc[T]) error {
	switch qc.Value.(type) {
	case string: // Plain SQL query case.
		qc.Querier = &inspect.SQLQuery[T]{
			SQL:   qc.Value.(string),
			RowTo: rowTo,
		}
	case []interface{}: // YAML values case.
		rawList := qc.Value.([]interface{})
		rows := make([]T, 0)
		for _, rawRow := range rawList {
			row, err := yamlTo(rawRow)
			if err != nil {
				return fmt.Errorf("bad value: %w", err)
			}
			rows = append(rows, row)
		}
		qc.Querier = &inspect.YAMLQuery[T]{
			Rows: rows,
		}
	default:
		return fmt.Errorf("bad query")
	}
	return nil
}

type YamlToFunc[T any] func(row interface{}) (T, error)

func YamlTo[T any](raw interface{}) (T, error) {
	return raw.(T), nil
}
