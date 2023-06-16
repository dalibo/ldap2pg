package sync

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slog"
)

func (wanted Wanted) DiffPrivileges(current []privilege.Grant, defaultDatabase string) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		wantedGrants := mapset.NewSet(wanted.Grants...)
		// Revoke spurious grants.
		for _, grant := range current {
			// Drop Grantor from inspected.
			grant.Grantor = ""
			if wantedGrants.Contains(grant) {
				continue
			}

			p := privilege.Map[grant.Target]
			slog.Debug("Revoke grant.", "target", grant.Target)
			sql, args := p.BuildRevoke(grant)
			q := postgres.SyncQuery{
				Description: "Revoke grant.",
				LogArgs: []interface{}{
					"grant", grant.Grantee,
				},
				Database:  grant.Database,
				Query:     sql,
				QueryArgs: args,
			}
			if "" == q.Database {
				q.Database = defaultDatabase
			}
			ch <- q
		}
	}()
	return ch
}
