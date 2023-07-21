package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// DBPoolMap track a single connection per database
type DBPoolMap map[string]*pgx.Conn

// Global DBPool.
var DBPool DBPoolMap

func init() {
	DBPool = make(DBPoolMap)
}

func (p DBPoolMap) Get(ctx context.Context, database string) (*pgx.Conn, error) {
	connp, ok := p[database]
	if ok {
		return connp, nil
	}
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.Database = database
	slog.Debug("Opening Postgres connection.", "database", config.Database)
	connp, err = pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	p[database] = connp
	return connp, nil
}

func (p DBPoolMap) CloseAll(ctx context.Context) {
	var names []string
	for name, connp := range p {
		slog.Debug("Closing Postgres connection.", "database", name)
		connp.Close(ctx)
		names = append(names, name)
	}
	for _, name := range names {
		delete(p, name)
	}
}
