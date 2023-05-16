// Configurable and overridable queries.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/jackc/pgx/v5"
	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

// INSPECT

func RunQuery[T any](q interface{}, pgconn *pgx.Conn, pgFun pgx.RowToFunc[T], yamlFun inspect.YamlToFunc[T]) <-chan any {
	ch := make(chan any)
	go func() {
		defer close(ch)
		var sql string
		rowsOrSQL, ok := q.(inspect.RowsOrSQL)
		if ok {
			if inspect.IsPredefined(rowsOrSQL) {
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

// SYNC

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

type FmtQueryRewriter struct{}

func (q FmtQueryRewriter) RewriteQuery(_ context.Context, conn *pgx.Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
	sql = strings.TrimSpace(dedent.Dedent(sql))
	var fmtArgs []interface{}
	for _, arg := range args {
		arg, err = formatArg(conn, arg)
		if err != nil {
			return
		}
		fmtArgs = append(fmtArgs, arg)
	}
	newSQL = fmt.Sprintf(sql, fmtArgs...)
	return
}

func formatArg(conn *pgx.Conn, arg interface{}) (newArg any, err error) {
	switch arg.(type) {
	case pgx.Identifier:
		newArg = arg.(pgx.Identifier).Sanitize()
	case string:
		s, err := conn.PgConn().EscapeString(arg.(string))
		if err != nil {
			return newArg, err
		}
		newArg = "'" + s + "'"
	case []interface{}:
		b := strings.Builder{}
		for _, item := range arg.([]interface{}) {
			item, err := formatArg(conn, item)
			if err != nil {
				return newArg, err
			}
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s", item))
		}
		newArg = b.String()
	default:
		newArg = arg
	}
	return
}
