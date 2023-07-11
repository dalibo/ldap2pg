package inspect

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
)

type Config struct {
	FallbackOwner       string
	DatabasesQuery      Querier[string]
	ManagedRolesQuery   Querier[string]
	RolesBlacklistQuery Querier[string]
	SchemasQuery        Querier[postgres.Schema]
	ManagedPrivileges   map[string][]string // SCHEMAS -> [USAGE, CREATE], ...
}
