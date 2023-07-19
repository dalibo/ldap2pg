package privilege

import (
	"context"
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type Inspector struct {
	dbmap             postgres.DBMap
	defaultDatabase   string
	managedPrivileges map[string][]string

	ctx        context.Context
	inspecters map[string]Inspecter
	grantChan  chan Grant
	err        error
	grant      Grant
}

func NewInspector(databases postgres.DBMap, defaultDatabase string, managedPrivileges map[string][]string) Inspector {
	return Inspector{
		dbmap:             databases,
		defaultDatabase:   defaultDatabase,
		managedPrivileges: managedPrivileges,
	}
}

func (i *Inspector) Run(ctx context.Context) {
	i.ctx = ctx
	i.inspecters = mapInspecters(maps.Keys(i.managedPrivileges))
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
	Databases(m postgres.DBMap, defaultDatabase string) []string
	Inspect() string
	RowTo(pgx.CollectableRow) (Grant, error)
}

func (i *Inspector) iterGrants() chan Grant {
	ch := make(chan Grant)
	go func() {
		defer close(ch)
		for object, p := range i.inspecters {
			arg, ok := i.managedPrivileges[object]
			if !ok {
				continue
			}

			for _, database := range p.Databases(i.dbmap, i.defaultDatabase) {
				slog.Debug("Inspecting grants.", "database", database, "object", p)
				pgconn, err := postgres.DBPool.Get(i.ctx, database)
				if err != nil {
					i.err = err
					return
				}

				sql := p.Inspect()
				slog.Debug("Executing SQL query:\n"+sql, "arg", arg)
				rows, err := pgconn.Query(i.ctx, sql, arg)
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
					grant.Database = database

					database, known := i.dbmap[grant.Database]
					if !known {
						continue
					}

					if "" != grant.Schema {
						_, known = database.Schemas[grant.Schema]
						if !known {
							continue
						}
					}

					grant.Normalize()

					ch <- grant
				}

			}
		}
	}()
	return ch
}

func mapInspecters(managedPrivileges []string) (out map[string]Inspecter) {
	out = make(map[string]Inspecter)
	var i Inspecter
	for k, p := range Builtins {
		if !slices.Contains(managedPrivileges, k) {
			continue
		}
		if "GLOBAL DEFAULT" == p.Object {
			i = NewGlobalDefault(p.Object, p.Inspect)
		} else if "SCHEMA DEFAULT" == p.Object {
			i = NewSchemaDefault(p.Object, p.Inspect)
		} else if strings.HasPrefix(p.Object, "ALL ") {
			i = NewAll(p.Object, p.Inspect)
		} else if "instance" == p.Scope {
			i = NewInstance(p.Object, p.Inspect)
		} else if "database" == p.Scope {
			i = NewDatabase(p.Object, p.Inspect)
		} else {
			continue
		}
		out[k] = i
	}
	return
}
