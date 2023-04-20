package postgres

import "github.com/jackc/pgx/v5"

type Database struct {
	Name  string
	Owner string
}

func RowToDatabase(row pgx.CollectableRow) (database Database, err error) {
	err = row.Scan(&database.Name, &database.Owner)
	return
}
