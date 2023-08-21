package privilege

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

type TypeMap map[string][]string

type Inspector struct {
	database          postgres.Database
	defaultDatabase   string
	managedPrivileges map[string][]string

	ctx       context.Context
	grantChan chan Grant
	err       error
	grant     Grant
}

func NewInspector(database postgres.Database, defaultDatabase string, managedPrivileges map[string][]string) Inspector {
	return Inspector{
		database:          database,
		defaultDatabase:   defaultDatabase,
		managedPrivileges: managedPrivileges,
	}
}

func (i *Inspector) Run(ctx context.Context) {
	i.ctx = ctx
	i.grantChan = i.iterGrants()
}

func (i *Inspector) Next() bool {
	grant, ok := <-i.grantChan
	if !ok {
		return false
	}
	if i.err != nil {
		return false
	}
	i.grant = grant
	return true
}

func (i Inspector) Grant() Grant {
	if i.err != nil {
		panic("inconsistent state")
	}
	return i.grant
}

func (i Inspector) Err() error {
	return i.err
}

type Inspecter interface {
	IsGlobal() bool
	Inspect() string
	RowTo(pgx.CollectableRow) (Grant, error)
}

func (i *Inspector) iterGrants() chan Grant {
	ch := make(chan Grant)
	go func() {
		defer close(ch)
		runGlobal := i.database.Name == i.defaultDatabase
		names := maps.Keys(Builtins)
		slices.Sort(names)
		for _, object := range names {
			arg, ok := i.managedPrivileges[object]
			if !ok {
				continue
			}

			p := Builtins[object]
			if p.IsGlobal() && !runGlobal {
				continue
			}

			slog.Debug("Inspecting grants.", "object", p, "database", i.database.Name)
			i.inspect1(object, p, arg, ch)
		}
	}()
	return ch
}

func (i *Inspector) inspect1(object string, p Privilege, types []string, ch chan Grant) {
	pgconn, err := postgres.GetConn(i.ctx, i.database.Name)
	if err != nil {
		i.err = err
		return
	}

	sql := p.Inspect()
	slog.Debug("Executing SQL query:\n"+sql, "arg", types)
	rows, err := pgconn.Query(i.ctx, sql, types)
	if err != nil {
		i.err = fmt.Errorf("bad query: %w", err)
		return
	}
	for rows.Next() {
		grant, err := p.RowTo(rows)
		if err != nil {
			i.err = fmt.Errorf("bad row: %w", err)
			return
		}
		grant.Database = i.database.Name

		if "" != grant.Schema {
			_, known := i.database.Schemas[grant.Schema]
			if !known {
				continue
			}
		}

		ch <- grant
	}
	if err := rows.Err(); err != nil {
		i.err = fmt.Errorf("%s: %w", object, err)
		return
	}
}
