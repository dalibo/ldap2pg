// Configurable and overridable queries.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

// INSPECT

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

func (q FmtQueryRewriter) RewriteQuery(ctx context.Context, conn *pgx.Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
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
