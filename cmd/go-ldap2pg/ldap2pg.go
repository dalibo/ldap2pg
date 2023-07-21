package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slog"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/role"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		slog.Error("Panic!", "err", r)
		buf := debug.Stack()
		fmt.Fprintf(os.Stderr, "%s", buf)
		slog.Error("Aborting ldap2pg.", "err", r)
		slog.Error("Please file an issue at https://github.com/dalibo/ldap2pg/issue/new with full log.")
		os.Exit(1)
	}()

	// Bootstrap logging first to log in setup.
	internal.SetLoggingHandler(slog.LevelInfo, isatty.IsTerminal(os.Stderr.Fd()))
	setupViper()
	if viper.GetBool("help") {
		pflag.Usage()
		return
	} else if viper.GetBool("version") {
		showVersion()
		return
	}
	err := ldap2pg(ctx)
	if err != nil {
		slog.Error("Fatal error.", "err", err)
		os.Exit(1)
	}
}

func ldap2pg(ctx context.Context) (err error) {
	defer postgres.DBPool.CloseAll(ctx)
	defer postgres.CloseConn(ctx)

	start := time.Now()

	controller, err := unmarshalController()
	if err != nil {
		return
	}

	internal.SetLoggingHandler(controller.LogLevel, controller.Color)
	slog.Info("Starting ldap2pg",
		"commit", internal.ShortRevision,
		"version", internal.Version,
		"runtime", runtime.Version())
	slog.Warn("go-ldap2pg is alpha software! Use at your own risks!")

	configPath := config.FindFile(controller.Config)
	slog.Info("Using YAML configuration file.", "path", configPath)
	c, err := config.Load(configPath)
	if err != nil {
		return
	}
	if controller.SkipPrivileges {
		c.DropPrivileges()
	}

	pc := c.Postgres.Build()
	instance, err := inspect.Stage0(ctx, pc)
	wantedRoles, wantedGrants, err := c.SyncMap.Run(&controller.LdapWatch, instance.RolesBlacklist, c.Privileges)
	if err != nil {
		return
	}

	// Describe instance, running user, find databases objects, roles, etc.
	err = instance.InspectStage1(ctx, pc)
	if err != nil {
		return
	}

	if controller.Real {
		slog.Info("Real mode. Postgres instance will modified.")
	} else {
		slog.Warn("Dry run. Postgres instance will be untouched.")
	}

	queries := role.Diff(instance.AllRoles, instance.ManagedRoles, wantedRoles, instance.Me, instance.FallbackOwner, &instance.Databases)
	queries = postgres.GroupByDatabase(instance.Databases, instance.DefaultDatabase, queries)
	stageCount, err := postgres.Apply(ctx, &controller.PostgresWatch, queries, controller.Real)
	if err != nil {
		return
	}
	if 0 == stageCount {
		slog.Info("All roles synchronized.")
	}
	queryCount := stageCount

	if c.ArePrivilegesManaged() {
		// Get the effective list of managed roles.
		managedRoles := maps.Keys(wantedRoles)
		_, ok := instance.ManagedRoles["public"]
		if ok {
			managedRoles = append(managedRoles, "public")
		}

		// Inspect grants, owners, etc.
		err = instance.InspectStage2(ctx, pc, managedRoles)
		if err != nil {
			return
		}
		grants := privilege.Expand(wantedGrants, instance.Databases)
		queries = privilege.Diff(instance.Grants, grants)
		queries = postgres.GroupByDatabase(instance.Databases, instance.DefaultDatabase, queries)
		stageCount, err = postgres.Apply(ctx, &controller.PostgresWatch, queries, controller.Real)
		if err != nil {
			return
		}
		if 0 == stageCount {
			slog.Info("All privileges synchronized.")
		}
		queryCount += stageCount

		err = instance.InspectStage3(ctx, managedRoles)
		if err != nil {
			return
		}
		grants = privilege.ExpandDefault(wantedGrants, instance.Databases)
		queries = privilege.DiffDefault(instance.Grants, grants)
		queries = postgres.GroupByDatabase(instance.Databases, instance.DefaultDatabase, queries)
		stageCount, err = postgres.Apply(ctx, &controller.PostgresWatch, queries, controller.Real)
		if err != nil {
			return
		}
		if 0 == stageCount {
			slog.Info("All default privileges configured.")
		}
		queryCount += stageCount
	} else {
		slog.Info("Not synchronizing privileges.")
	}

	vmPeak := perf.ReadVMPeak()
	elapsed := time.Since(start)
	logAttrs := []interface{}{
		"elapsed", elapsed,
		"mempeak", perf.FormatBytes(vmPeak),
		"postgres", controller.PostgresWatch.Total,
		"queries", queryCount,
		"ldap", controller.LdapWatch.Total,
		"searches", controller.LdapWatch.Count,
	}
	if queryCount > 0 {
		slog.Info("Comparison complete.", logAttrs...)
	} else {
		slog.Info("Nothing to do.", logAttrs...)
	}

	if controller.Check && queryCount > 0 {
		os.Exit(1)
	}

	return
}

func showVersion() {
	fmt.Printf("go-ldap2pg %s\n", internal.Version)

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	modmap := make(map[string]string)
	for _, mod := range bi.Deps {
		modmap[mod.Path] = mod.Version
	}
	modules := []string{
		"github.com/jackc/pgx/v5",
		"github.com/go-ldap/ldap/v3",
		"gopkg.in/yaml.v3",
	}
	for _, mod := range modules {
		fmt.Printf("%s %s\n", mod, modmap[mod])
	}

	fmt.Printf("%s %s %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
