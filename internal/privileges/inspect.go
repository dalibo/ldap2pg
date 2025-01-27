package privileges

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
)

// InspectGrants returns ACL items from Postgres instance.
func InspectGrants(ctx context.Context, db postgres.Database, acl string, roles mapset.Set[string]) (out []Grant, err error) {
	inspector := newInspector(db, acl)
	for inspector.Run(ctx); inspector.Next(); {
		grant := inspector.Grant()
		if grant.IsRelevant() && !roles.Contains(grant.Grantee) {
			continue
		}
		if grant.IsDefault() && !roles.Contains(grant.Owner) {
			continue
		}

		grant.Normalize()
		// Special case: ignore database grants on unmanaged databases.
		if "DATABASE" == grant.ACLName() {
			_, exists := postgres.Databases[grant.Object]
			if !exists {
				continue
			}
		}

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
	database postgres.Database
	acl      string

	ctx       context.Context
	grantChan chan Grant
	err       error
	grant     Grant
}

func newInspector(database postgres.Database, acl string) inspector {
	return inspector{
		database: database,
		acl:      acl,
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
		acl := acls[i.acl]
		types := managedACLs[i.acl]
		slog.Debug("Inspecting grants.", "acl", i.acl, "database", i.database.Name)
		pgconn, err := postgres.GetConn(i.ctx, i.database.Name)
		if err != nil {
			i.err = err
			return
		}

		sql := acl.Inspect()
		slog.Debug("Executing SQL query:\n"+sql, "arg", types)
		rows, err := pgconn.Query(i.ctx, sql, types)
		if err != nil {
			i.err = fmt.Errorf("bad query: %w", err)
			return
		}
		for rows.Next() {
			grant, err := acl.RowTo(rows)
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
			i.err = fmt.Errorf("%s: %w", i.acl, err)
			return
		}
	}()
	return ch
}
