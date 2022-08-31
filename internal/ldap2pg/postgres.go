package ldap2pg

import (
	"context"

	"github.com/jackc/pgx/v4"
)

func PostgresConnect(config Config) error {
	Logger.Infow("Connecting to PostgreSQL instance.")
	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return err
	}
	defer pgconn.Close(ctx)

	var me string
	err = pgconn.QueryRow(ctx, "SELECT CURRENT_USER;").Scan(&me)
	if err != nil {
		return err
	}

	Logger.Debugw("Introspected PostgreSQL user.", "username", me)
	return nil
}
