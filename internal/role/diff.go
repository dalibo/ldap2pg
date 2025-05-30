package role

import (
	"log/slog"

	"github.com/dalibo/ldap2pg/v6/internal/postgres"
)

func Diff(all, managed, wanted Map, fallbackOwner string) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing roles.
		for _, name := range wanted.Flatten() {
			role := wanted[name]
			if other, ok := all[name]; ok {
				// Check for existing role, even if unmanaged.
				if _, ok := managed[name]; !ok {
					slog.Warn("Reusing unmanaged role. Ensure managed_roles_query returns all wanted roles.", "role", name)
				}
				sendQueries(other.Alter(role), ch)
			} else {
				sendQueries(role.Create(), ch)
			}
		}

		// Drop spurious roles.
		// Only from managed roles.
		for name := range managed {
			if _, ok := wanted[name]; ok {
				continue
			}

			if name == "public" {
				continue
			}

			role, ok := all[name]
			if !ok {
				// Already dropped. ldap2pg hits this case whan
				// ManagedRoles is static.
				continue
			}

			sendQueries(role.Drop(fallbackOwner), ch)
		}
	}()
	return ch
}

func sendQueries(queries []postgres.SyncQuery, ch chan postgres.SyncQuery) {
	for _, q := range queries {
		ch <- q
	}
}
