package inspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Querier abstracts the execution of a SQL query or the copy of static rows
// from YAML.
type Querier[T any] interface {
	Query(context.Context, Conn)
	Next() bool
	Err() error
	Row() T
}

// Conn allows to inject a mock.
type Conn interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// YAMLQuery holds a static rowset from config file
type YAMLQuery[T any] struct {
	Rows         []T
	currentIndex int
}

func (q *YAMLQuery[_]) Query(_ context.Context, _ Conn) {
	q.currentIndex = -1
	slog.Debug("Reading values from YAML.")
}

func (q *YAMLQuery[_]) Next() bool {
	q.currentIndex++
	return q.currentIndex < len(q.Rows)
}

func (q *YAMLQuery[_]) Err() error {
	return nil
}

func (q *YAMLQuery[T]) Row() T {
	return q.Rows[q.currentIndex]
}

// SQLQuery holds a configurable SQL query en handle fetching rows from
// Postgres.
// *SQLQuery implements Querier.
type SQLQuery[T any] struct {
	SQL   string
	RowTo pgx.RowToFunc[T]

	rows pgx.Rows
	err  error
	row  T
}

func (q *SQLQuery[_]) Query(ctx context.Context, pgconn Conn) {
	slog.Debug("Executing SQL query:\n" + q.SQL)
	rows, err := pgconn.Query(ctx, q.SQL)
	if err != nil {
		q.err = fmt.Errorf("bad query: %w", err)
		return
	}
	q.rows = rows
}

func (q *SQLQuery[_]) Next() bool {
	if q.err != nil {
		return false
	}
	next := q.rows.Next()
	if !next {
		return false
	}
	q.err = q.rows.Err()
	if q.err != nil {
		return false
	}
	q.row, q.err = q.RowTo(q.rows)
	if q.err != nil {
		return false
	}
	return next
}

func (q *SQLQuery[_]) Err() error {
	return q.err
}

func (q *SQLQuery[T]) Row() T {
	return q.row
}
