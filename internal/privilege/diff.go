package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
)

type Granter interface {
	Grant(Grant) postgres.SyncQuery
}

type Revoker interface {
	Revoke(Grant) postgres.SyncQuery
}

func Diff(current, wanted []Grant) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		wantedSet := mapset.NewSet(wanted...)
		// Revoke spurious grants.
		for _, grant := range current {
			if grant.IsDefault() {
				continue
			}

			wantedGrant := grant
			// Always search a full grant in wanted. If we have a
			// partial grant in instance, it will be regranted in
			// grant loop.
			wantedGrant.Partial = false
			if wantedSet.Contains(wantedGrant) {
				continue
			}

			if "" == grant.Type {
				// Don't revoke irrelevant ANY ... IN SCHEMA
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
			if grant.IsDefault() {
				continue
			}

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

func DiffDefault(current, wanted []Grant) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		wantedSet := mapset.NewSet(wanted...)
		// Revoke spurious grants.
		for _, grant := range current {
			if !grant.IsDefault() {
				continue
			}

			if wantedSet.Contains(grant) {
				continue
			}

			p := grant.Privilege()
			q := p.Revoke(grant)
			q.Description = "Revoke default privilege."
			q.Database = grant.Database
			q.LogArgs = []interface{}{"grant", grant}
			ch <- q
		}

		currentSet := mapset.NewSet(current...)
		for _, grant := range wanted {
			if !grant.IsDefault() {
				continue
			}

			if currentSet.Contains(grant) {
				continue
			}

			p := grant.Privilege()
			q := p.Grant(grant)
			q.Description = "Grant default privilege."
			q.Database = grant.Database
			q.LogArgs = []interface{}{"grant", grant}
			ch <- q
		}
	}()
	return ch
}
