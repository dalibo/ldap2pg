package privileges

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
)

// Inspect returns ACL items from Postgres instance.
func Inspect(ctx context.Context, db postgres.Database, acl string, roles mapset.Set[string]) (out []Grant, err error) {
	inspector := inspector{database: db, acl: acl}
	for inspector.Run(ctx); inspector.Next(); {
		grant := inspector.Grant()
		// Drop wildcard on public if public is not managed.
		if grant.IsWildcard() && !roles.Contains(grant.Grantee) {
			continue
		}
		if grant.IsDefault() && !roles.Contains(grant.Owner) {
			continue
		}

		// Special case: ignore database grants on unmanaged databases.
		if "DATABASE" == grant.ACLName() {
			_, exists := postgres.Databases[grant.Object]
			if !exists {
				continue
			}
		}

		slog.Debug("Found grant in Postgres instance.", "grant", grant, "database", grant.Database)
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

func (i *inspector) iterGrants() chan Grant {
	ch := make(chan Grant)
	go func() {
		defer close(ch)
		acl := aclImplentations[i.acl]
		sql := acls[i.acl].Inspect
		types := managedACLs[i.acl]
		slog.Debug("Inspecting grants.", "acl", i.acl, "database", i.database.Name)
		pgconn, err := postgres.GetConn(i.ctx, i.database.Name)
		if err != nil {
			i.err = err
			return
		}

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
