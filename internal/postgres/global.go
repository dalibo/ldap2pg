package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	globalConn *pgx.Conn
	globalConf *pgx.ConnConfig
)

func Configure(dsn string) (err error) {
	globalConf, err = pgx.ParseConfig(dsn)
	if globalConf.ConnectTimeout == 0 {
		slog.Debug("Setting default Postgres connection timeout.", "timeout", "5s")
		globalConf.ConnectTimeout, _ = time.ParseDuration("5s")
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
