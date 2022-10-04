package ldap2pg

import (
	"context"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
)

func PostgresConnect(config Config) error {
	log.Info("Connecting to PostgreSQL instance.")
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

	log.
		WithField("username", me).
		Debug("Introspected PostgreSQL user.")
	return nil
}
