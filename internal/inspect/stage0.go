package inspect

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

// Fourzitou struct holding everything need to synchronize Instance.
type Instance struct {
	AllRoles         role.Map
	Databases        postgres.DBMap
	DefaultDatabase  string
	FallbackOwner    string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     role.Map
	Me               role.Role
	RolesBlacklist   lists.Blacklist
	Grants           []privilege.Grant
}

func Stage0(ctx context.Context, rolesBlackListQ Querier[string]) (instance Instance, err error) {
	slog.Debug("Stage 0: role blacklist.")
	instance = Instance{}

	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		return
	}
	defer pgconn.Close(ctx)

	slog.Debug("Inspecting roles blacklist.")
	for rolesBlackListQ.Query(ctx, pgconn); rolesBlackListQ.Next(); {
		instance.RolesBlacklist = append(instance.RolesBlacklist, rolesBlackListQ.Row())
	}
	if err := rolesBlackListQ.Err(); err != nil {
		return instance, fmt.Errorf("roles_blacklist_query: %w", err)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

	return
}
