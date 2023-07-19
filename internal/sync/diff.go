package sync

import (
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/role"
	"golang.org/x/exp/slog"
)

func DiffRoles(instance inspect.Instance, wanted role.Map) <-chan postgres.SyncQuery {
	ch := make(chan postgres.SyncQuery)
	go func() {
		defer close(ch)
		// Create missing roles.
		for _, name := range wanted.Flatten() {
			role := wanted[name]
			if other, ok := instance.AllRoles[name]; ok {
				// Check for existing role, even if unmanaged.
				if _, ok := instance.ManagedRoles[name]; !ok {
					slog.Warn("Reusing unmanaged role. Ensure managed_roles_query returns all wanted roles.", "role", name)
				}
				sendQueries(other.Alter(role), ch, instance.DefaultDatabase)
			} else {
				sendQueries(role.Create(), ch, instance.DefaultDatabase)
			}
		}

		// Drop spurious roles.
		// Only from managed roles.
		for name := range instance.ManagedRoles {
			if _, ok := wanted[name]; ok {
				continue
			}

			if "public" == name {
				continue
			}

			role, ok := instance.AllRoles[name]
			if !ok {
				// Already dropped. ldap2pg hits this case whan
				// ManagedRoles is static.
				continue
			}

			sendQueries(role.Drop(&instance.Databases, instance.Me, instance.FallbackOwner), ch, instance.DefaultDatabase)
		}
	}()
	return ch
}

func sendQueries(queries []postgres.SyncQuery, ch chan postgres.SyncQuery, defaultDatabase string) {
	for _, q := range queries {
		if "" == q.Database {
			q.Database = defaultDatabase
		}
		ch <- q
	}
}
