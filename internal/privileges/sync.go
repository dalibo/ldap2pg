package privileges

import (
	"context"

	"github.com/dalibo/ldap2pg/internal/postgres"
	mapset "github.com/deckarep/golang-set/v2"
)

func Sync(ctx context.Context, really bool, dbname string, current, wanted []Grant) (int, error) {
	wanted = Expand(wanted, postgres.Databases[dbname])
	queries := diff(current, wanted)
	return postgres.Apply(ctx, queries, really)
}

func diff(current, wanted []Grant) <-chan postgres.SyncQuery {
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

			q := grant.FormatQuery(acls[grant.Target].Revoke)
			q.Description = "Revoke privileges."
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
			wildcardGrant := grant
			wildcardGrant.Grantee = "public"
			wildcardGrant.Type = ""
			if currentSet.Contains(wildcardGrant) {
				continue
			}

			q := grant.FormatQuery(acls[grant.Target].Grant)
			q.Description = "Grant privileges."
			q.Database = grant.Database
			q.LogArgs = []interface{}{"grant", grant}
			ch <- q
		}
	}()
	return ch
}
