package inspect

type Config struct {
	FallbackOwner       string
	DatabasesQuery      Querier[string]
	ManagedRolesQuery   Querier[string]
	RolesBlacklistQuery Querier[string]
}
