package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Database struct {
	Name    string
	Owner   string
	Schemas map[string]Schema
}
type DBMap map[string]Database

func RowToDatabase(row pgx.CollectableRow) (database Database, err error) {
	err = row.Scan(&database.Name, &database.Owner)
	database.Schemas = make(map[string]Schema)
	return
}

type Schema struct {
	Name     string
	Owner    string
	Creators []string
}

func RowToSchema(row pgx.CollectableRow) (s Schema, err error) {
	switch len(row.FieldDescriptions()) {
	case 1:
		err = row.Scan(&s.Name)
	case 2:
		err = row.Scan(&s.Name, &s.Owner)
	default:
		err = fmt.Errorf("wrong number of returned columns")
	}
	return
}
