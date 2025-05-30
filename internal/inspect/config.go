package inspect

import (
	"github.com/dalibo/ldap2pg/v6/internal/postgres"
)

type Config struct {
	FallbackOwner       string
	DatabasesQuery      Querier[string]
	ManagedRolesQuery   Querier[string]
	RolesBlacklistQuery Querier[string]
	SchemasQuery        Querier[postgres.Schema]
}
