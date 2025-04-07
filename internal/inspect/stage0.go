package inspect

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slices"
)

// Fourzitou struct holding everything need to synchronize Instance.
type Instance struct {
	AllRoles         role.Map
	DefaultDatabase  string
	FallbackOwner    string
	ManagedDatabases mapset.Set[string]
	ManagedRoles     role.Map
	Me               role.Role
	RolesBlacklist   lists.Blacklist
}

func Stage0(ctx context.Context, pc Config) (instance Instance, err error) {
	slog.Debug("Stage 0: role blacklist.")
	instance = Instance{}

	err = instance.InspectSession(ctx, pc.FallbackOwner)
	if err != nil {
		return instance, fmt.Errorf("session: %w", err)
	}

	slog.Debug("Inspecting roles blacklist.", "config", "roles_blacklist_query")
	conn, err := postgres.GetConn(ctx, "")
	if err != nil {
		return instance, err
	}

	for pc.RolesBlacklistQuery.Query(ctx, conn); pc.RolesBlacklistQuery.Next(); {
		instance.RolesBlacklist = append(instance.RolesBlacklist, pc.RolesBlacklistQuery.Row())
	}
	if err := pc.RolesBlacklistQuery.Err(); err != nil {
		return instance, fmt.Errorf("roles_blacklist_query: %w", err)
	}
	if !slices.Contains(instance.RolesBlacklist, instance.Me.Name) {
		slog.Debug("Blacklisting self.")
		instance.RolesBlacklist = append(instance.RolesBlacklist, instance.Me.Name)
	}
	err = instance.RolesBlacklist.Check()
	if err != nil {
		return instance, fmt.Errorf("roles_blacklist_query: %w", err)
	}
	slog.Debug("Roles blacklist loaded.", "patterns", instance.RolesBlacklist)

	return
}

func (instance *Instance) InspectSession(ctx context.Context, fallbackOwner string) error {
	pgconn, err := postgres.GetConn(ctx, "")
	if err != nil {
		return err
	}

	slog.Debug("Inspecting PostgreSQL server and session.")
	slog.Debug("Executing SQL query:\n" + sessionQuery)
	var rows pgx.Rows
	Watch.TimeIt(func() {
		rows, err = pgconn.Query(ctx, sessionQuery)
	})
	if err != nil {
		return err
	}
	if !rows.Next() {
		panic("No data returned.")
	}
	var clusterName, serverVersion string
	var serverVersionNum int
	err = rows.Scan(
		&serverVersion, &serverVersionNum,
		&clusterName, &instance.DefaultDatabase,
		&instance.Me.Name, &instance.Me.Options.Super,
	)
	if err != nil {
		return err
	}

	var msg string
	if instance.Me.Options.Super {
		msg = "Running as superuser."
	} else if serverVersionNum < 160000 {
		slog.Warn("Running as unprivileged user on Postgres 15 and lower.", "version", serverVersion)
		slog.Warn("Unprivileged user is flawed before Postgres 16.")
		slog.Warn("Upgrade to Postgres 16 or later, switch to superuser or stick to ldap2pg 6.0.")
		return fmt.Errorf("unprivileged user on pre-16 Postgres")
	} else {
		msg = "Running as unprivileged user."
	}
	slog.Info(
		msg,
		"user", instance.Me.Name,
		"super", instance.Me.Options.Super,
		"server", serverVersion,
		"cluster", clusterName,
		"database", instance.DefaultDatabase,
	)
	if rows.Next() {
		panic("Multiple row returned.")
	}
	if fallbackOwner == "" {
		instance.FallbackOwner = instance.Me.Name
	} else {
		instance.FallbackOwner = fallbackOwner
	}
	slog.Debug("Fallback owner configured.", "role", instance.FallbackOwner)

	return nil
}
