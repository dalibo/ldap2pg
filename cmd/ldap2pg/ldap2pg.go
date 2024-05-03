package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer logPanic()

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
		if internal.CurrentLevel > slog.LevelDebug {
			slog.Error("Run ldap2pg with --verbose to get more informations.")
		}
		os.Exit(1)
	}
}

func ldap2pg(ctx context.Context) (err error) {
	stop, err := startProfiling()
	if err != nil {
		return
	}
	if stop != nil {
		defer stop()
	}
	defer postgres.CloseConn(ctx)

	start := time.Now()

	controller, err := unmarshalController()
	if err != nil {
		return
	}

	internal.SetLoggingHandler(controller.LogLevel, controller.Color)
	slog.Info("Starting ldap2pg",
		"version", version,
		"runtime", runtime.Version(),
		"commit", commit,
		"pid", os.Getpid(),
	)

	if strings.Contains(version, "-") {
		slog.Warn("Running a prerelease! Use at your own risks!")
	}

	err = changeDirectory(controller.Directory)
	if err != nil {
		return
	}

	err = postgres.Configure(controller.Dsn)
	if err != nil {
		return
	}

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
	if err != nil {
		return
	}
	wantedRoles, wantedGrants, err := c.SyncMap.Run(&controller.LdapWatch, instance.RolesBlacklist, c.Privileges)
	if err != nil {
		return
	}
	for _, g := range wantedGrants {
		_, ok := wantedRoles[g.Grantee]
		if !ok {
			slog.Info("Generated grant on unwanted role.", "grant", g)
			return fmt.Errorf("grant on unwanted role")
		}
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

	queries := role.Diff(instance.AllRoles, instance.ManagedRoles, wantedRoles, instance.FallbackOwner, &instance.Databases)
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
		slog.Debug("Synchronizing privileges.")
		// Get the effective list of managed roles.
		managedRoles := mapset.NewSet(maps.Keys(wantedRoles)...)
		_, ok := instance.ManagedRoles["public"]
		if ok {
			managedRoles.Add("public")
		}

		instancePrivileges, objectPrivileges, defaultPrivileges := c.Postgres.PrivilegesMap.BuildTypeMaps()

		// Start by default database. This allow to reuse the last
		// connexion openned when synchronizing roles.
		for _, dbname := range instance.Databases.SyncOrder(instance.DefaultDatabase, true) {
			slog.Debug("Stage 2: privileges.", "database", dbname)
			err := instance.InspectStage2(ctx, dbname, pc.SchemasQuery)
			if err != nil {
				return fmt.Errorf("inspect: %w", err)
			}
			var privileges privilege.TypeMap
			if dbname == instance.DefaultDatabase {
				slog.Debug("Managing instance wide privileges.", "database", dbname)
				privileges = make(privilege.TypeMap)
				maps.Copy(privileges, instancePrivileges)
				maps.Copy(privileges, objectPrivileges)
			} else {
				privileges = objectPrivileges
			}
			stageCount, err := syncPrivileges(ctx, &controller, &instance, managedRoles, wantedGrants, dbname, privileges)
			if err != nil {
				return fmt.Errorf("stage 2: %w", err)
			}
			if 0 == stageCount {
				slog.Info("All privileges configured.", "database", dbname)
			}
			queryCount += stageCount

			slog.Debug("Stage 3: default privileges.")
			err = instance.InspectStage3(ctx, dbname, managedRoles)
			if err != nil {
				return fmt.Errorf("inspect: %w", err)
			}
			stageCount, err = syncPrivileges(ctx, &controller, &instance, managedRoles, wantedGrants, dbname, defaultPrivileges)
			if err != nil {
				return fmt.Errorf("stage 3: %w", err)
			}
			if 0 == stageCount {
				slog.Info("All default privileges configured.", "database", dbname)
			}
			queryCount += stageCount
		}
	} else {
		slog.Debug("Not synchronizing privileges.")
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
	fmt.Printf("ldap2pg %s\n", version)

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

func syncPrivileges(ctx context.Context, controller *Controller, instance *inspect.Instance, roles mapset.Set[string], wantedGrants []privilege.Grant, dbname string, privileges privilege.TypeMap) (int, error) {
	stageCount := 0
	allDatabases := maps.Keys(instance.Databases)
	privKeys := maps.Keys(privileges)
	slices.Sort(privKeys)
	for _, priv := range privKeys {
		privileges := privilege.TypeMap{priv: privileges[priv]}
		expandedGrants := privilege.Expand(wantedGrants, privileges, instance.Databases[dbname], allDatabases)
		currentGrants, err := instance.InspectGrants(ctx, dbname, privileges, roles)
		// Special case, ignore grants on unmanaged databases.
		currentGrants = lists.Filter(currentGrants, func(g privilege.Grant) bool {
			if "DATABASE" != g.PrivilegeKey() {
				return true
			}
			_, ok := instance.Databases[g.Object]
			return ok
		})

		if err != nil {
			return 0, fmt.Errorf("privileges: %w", err)
		}
		queries := privilege.Diff(currentGrants, expandedGrants)
		count, err := postgres.Apply(ctx, &controller.PostgresWatch, queries, controller.Real)
		if err != nil {
			return 0, fmt.Errorf("apply: %w", err)
		}
		slog.Debug("Privilege synchronized.", "privilege", priv, "database", dbname)
		stageCount += count
	}
	return stageCount, nil
}

func logPanic() {
	r := recover()
	if r == nil {
		return
	}
	slog.Error("Panic!", "err", r)
	buf := debug.Stack()
	fmt.Fprintf(os.Stderr, "%s", buf)
	slog.Error("Aborting ldap2pg.", "err", r)
	if internal.CurrentLevel > slog.LevelDebug {
		slog.Error("Run ldap2pg with --verbose to get more informations.")
	}
	slog.Error("Please file an issue at https://github.com/dalibo/ldap2pg/issue/new with verbose log.")
	os.Exit(1)
}

func startProfiling() (stop func(), err error) {
	if !slices.Contains(os.Environ(), "CPUPROFILE=1") {
		return
	}
	slog.Debug("Starting CPU profiling.")
	f, err := os.Create("default.pgo")
	if err != nil {
		return
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		f.Close()
		return
	}
	stop = func() {
		slog.Debug("Stopping profiling.")
		pprof.StopCPUProfile()
		f.Close()
	}
	return
}

func changeDirectory(directory string) (err error) {
	if directory == "" {
		return
	}
	slog.Debug("Changing directory.", "path", directory)
	return os.Chdir(directory)
}
