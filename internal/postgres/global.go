package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

var (
	DefaultDatabase string
	globalConn      *pgx.Conn
)

func GetConn(ctx context.Context, database string) (*pgx.Conn, error) {
	if "" == database {
		database = DefaultDatabase
	}

	if nil != globalConn {
		c := globalConn.Config()
		if database != c.Database {
			CloseConn(ctx)
		}
	}

	if nil == globalConn {
		slog.Debug("Opening Postgres global connection.", "database", database)
		c, err := pgx.ParseConfig("connect_timeout=5")
		if err != nil {
			return nil, err
		}
		c.Database = database
		globalConn, err = pgx.ConnectConfig(ctx, c)
		if err != nil {
			return nil, err
		}

		version := globalConn.PgConn().ParameterStatus("server_version")
		if !strings.HasPrefix(version, "9.") && version >= "16" {
			slog.Debug("Configuring createrole_self_grant.", "server_version", version)
			_, err := globalConn.Exec(ctx, "SET createrole_self_grant TO inherit")
			if err != nil {
				return nil, err
			}
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
