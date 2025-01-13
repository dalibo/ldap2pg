package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	globalConn *pgx.Conn
	globalConf *pgx.ConnConfig
)

func Configure(dsn string) (err error) {
	globalConf, err = pgx.ParseConfig(dsn)
	if err != nil {
		return
	}
	if globalConf.ConnectTimeout == 0 {
		slog.Debug("Setting default Postgres connection timeout.", "timeout", "5s")
		globalConf.ConnectTimeout, _ = time.ParseDuration("5s")
		globalConf.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
			switch n.Severity {
			case "NOTICE":
				slog.Info("Postgres message.", "message", n.Message, "hint", n.Hint, "detail", n.Detail)
			case "WARNING":
				slog.Warn("Postgres warning.", "message", n.Message, "hint", n.Hint, "detail", n.Detail)
			case "ERROR", "FATAL", "PANIC":
				slog.Error("Postgres error.", "message", n.Message, "hint", n.Hint, "detail", n.Detail)
				panic("Postgres out of band error.") // We should propagate error. No case found yet.
			default:
				slog.Debug("Postgres message.", "message", n.Message, "severity", n.Severity, "detail", n.Detail)
			}
		}
	}
	return
}

func GetConn(ctx context.Context, database string) (*pgx.Conn, error) {
	if database == "" {
		database = globalConf.Database
	}

	if nil != globalConn {
		c := globalConn.Config()
		if database != c.Database {
			CloseConn(ctx)
		}
	}

	if nil == globalConn {
		var err error
		slog.Debug("Opening Postgres global connection.", "database", database)
		c := globalConf.Copy()
		c.Database = database
		globalConn, err = pgx.ConnectConfig(ctx, c)
		if err != nil {
			return nil, err
		}
	}

	return globalConn, nil
}

func CloseConn(ctx context.Context) {
	if nil == globalConn {
		return
	}
	c := globalConn.Config()
	slog.Debug("Closing Postgres global connection.", "database", c.Database)

	globalConn.Close(ctx)
	globalConn = nil
}
