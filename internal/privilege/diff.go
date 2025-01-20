package privilege

import (
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
)

type granter interface {
	Grant(Grant) postgres.SyncQuery
}

type revoker interface {
	Revoke(Grant) postgres.SyncQuery
}

func Diff(current, wanted []Grant) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		wantedSet := mapset.NewSet(wanted...)
		// Revoke spurious grants.
		for _, grant := range current {
			wantedGrant := grant
			// Always search a full grant in wanted. If we have a
			// partial grant in instance, it will be regranted in
			// grant loop.
			wantedGrant.Partial = false
			// Don't revoke irrelevant ANY ... IN SCHEMA
			if wantedSet.Contains(wantedGrant) || "" == grant.Type {
				continue
			}

			p := grant.Privilege()
			q := p.Revoke(grant)
			q.Description = "Revoke privilege."
			q.Database = grant.Database
			q.LogArgs = []interface{}{"grant", grant}
			ch <- q
		}

		currentSet := mapset.NewSet(current...)
		for _, grant := range wanted {
			if currentSet.Contains(grant) {
				continue
			}

			// Test if a GRANT ON ALL ... IN SCHEMA is irrelevant.
			// To avoid regranting each run.
			irrelevantGrant := grant
			irrelevantGrant.Grantee = "public"
			irrelevantGrant.Type = ""
			if currentSet.Contains(irrelevantGrant) {
				continue
			}

			slog.Debug("Wants grant.", "grant", grant)
			p := grant.Privilege()
			q := p.Grant(grant)
			q.Description = "Grant privilege."
			q.Database = grant.Database
			q.LogArgs = []interface{}{"grant", grant}
			ch <- q
		}
	}()
	return ch
}
