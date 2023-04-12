package internal

import (
	"context"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

type PostgresDBPool map[string]*pgx.Conn

func (p PostgresDBPool) Get(database string) (*pgx.Conn, error) {
	connp, ok := p[database]
	if ok {
		return connp, nil
	}
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.Database = database
	slog.Debug("Opening Postgres connection.", "db", config.Database)
	connp, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	p[database] = connp
	return connp, nil
}

func (p PostgresDBPool) CloseAll() {
	var names []string
	for name, connp := range p {
		slog.Debug("Closing Postgres connection.", "db", name)
		connp.Close(context.Background())
		names = append(names, name)
	}
	for _, name := range names {
		delete(p, name)
	}
}
