package privileges

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

// TypeMap lists managed privilege types for each ACL
//
// e.g.: SELECT, UPDATE for TABLES, EXECUTE for FUNCTIONS, etc.
type TypeMap map[string][]string

// InspectGrants returns ACL items from Postgres instance.
func InspectGrants(ctx context.Context, db postgres.Database, privs TypeMap, roles mapset.Set[string]) (out []Grant, err error) {
	inspector := newInspector(db, privs)
	for inspector.Run(ctx); inspector.Next(); {
		grant := inspector.Grant()
		if grant.IsRelevant() && !roles.Contains(grant.Grantee) {
			continue
		}
		if grant.IsDefault() && !roles.Contains(grant.Owner) {
			continue
		}

		grant.Normalize()

		slog.Debug("Found grant in Postgres instance.", "grant", grant)
		out = append(out, grant)
	}
	err = inspector.Err()
	return
}

// inspector orchestrates privilege inspection
//
// Delegates querying and scanning to ACL.
type inspector struct {
	database          postgres.Database
	managedPrivileges map[string][]string

	ctx       context.Context
	grantChan chan Grant
	err       error
	grant     Grant
}

func newInspector(database postgres.Database, managedPrivileges TypeMap) inspector {
	if len(managedPrivileges) > 1 {
		panic("only one ACL is supported")
	}
	return inspector{
		database:          database,
		managedPrivileges: managedPrivileges,
	}
}

func (i *inspector) Run(ctx context.Context) {
	i.ctx = ctx
	i.grantChan = i.iterGrants()
}

func (i *inspector) Next() bool {
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

func (i inspector) Grant() Grant {
	if i.err != nil {
		panic("inconsistent state")
	}
	return i.grant
}

func (i inspector) Err() error {
	return i.err
}

// Implemented by ACL types
//
// e.g. datacl, nspacl, etc.
type inspecter interface {
	IsGlobal() bool
	Inspect() string
	RowTo(pgx.CollectableRow) (Grant, error)
}

func (i *inspector) iterGrants() chan Grant {
	ch := make(chan Grant)
	go func() {
		defer close(ch)
		for name, types := range i.managedPrivileges {
			acl := acls[name]
			slog.Debug("Inspecting grants.", "acl", acl, "database", i.database.Name)
			i.inspect1(name, acl, types, ch)
		}
	}()
	return ch
}

func (i *inspector) inspect1(object string, p acl, types []string, ch chan Grant) {
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
