// Configurable and overridable queries.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/lithammer/dedent"
)

// SYNC

type SyncQuery struct {
	Description string
	LogArgs     []any
	Database    string
	Query       string
	QueryArgs   []any
}

func (q SyncQuery) IsZero() bool {
	return q.Query == ""
}

func (q SyncQuery) String() string {
	return q.Description
}

type FmtQueryRewriter struct{}

func (q FmtQueryRewriter) RewriteQuery(_ context.Context, conn *pgx.Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
	sql = strings.TrimSpace(dedent.Dedent(sql))
	var fmtArgs []any
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

func formatArg(conn *pgx.Conn, arg any) (newArg any, err error) {
	switch arg.(type) {
	case pgx.Identifier:
		newArg = arg.(pgx.Identifier).Sanitize()
	case string:
		s, err := conn.PgConn().EscapeString(arg.(string))
		if err != nil {
			return newArg, err
		}
		newArg = "'" + s + "'"
	case []any:
		b := strings.Builder{}
		for _, item := range arg.([]any) {
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

func GroupByDatabase(defaultDatabase string, in <-chan SyncQuery) chan SyncQuery {
	ch := make(chan SyncQuery)
	go func() {
		defer close(ch)
		var queries []SyncQuery
		databases := SyncOrder(defaultDatabase, false)

		for q := range in {
			if "<first>" == q.Database {
				q.Database = databases[0]
			} else if "" == q.Database {
				q.Database = defaultDatabase
			}
			queries = append(queries, q)
		}

		for _, name := range databases {
			for _, q := range queries {
				if q.Database == name {
					ch <- q
				}
			}
		}
	}()
	return ch
}
