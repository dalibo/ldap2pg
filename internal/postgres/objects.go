package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Database struct {
	Name    string
	Owner   string
	Schemas map[string]Schema
}

type DBMap map[string]Database

var Databases = make(DBMap)

func SyncOrder(defaultName string, defaultFirst bool) (out []string) {
	m := Databases
	names := maps.Keys(m)
	slices.Sort(names)
	_, ok := m[defaultName]
	if defaultFirst && ok {
		out = append(out, defaultName)
	}
	for _, name := range names {
		if defaultName != name {
			out = append(out, name)
		}
	}

	if !defaultFirst && ok {
		out = append(out, defaultName)
	}
	return
}

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

func YamlToSchema(in any) (out Schema, err error) {
	var ok bool
	out.Name, ok = in.(string)
	if !ok {
		panic("Unsupported schema value.")
	}
	return
}
