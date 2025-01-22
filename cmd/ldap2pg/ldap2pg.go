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
	"github.com/dalibo/ldap2pg/internal/errorlist"
	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer logPanic()

	// Bootstrap logging first to log in setup.
	internal.SetLoggingHandler(slog.LevelInfo, isatty.IsTerminal(os.Stderr.Fd()))
	loadEnvAndFlags()
	if k.Bool("help") {
		pflag.Usage()
		return
	} else if k.Bool("version") {
		showVersion()
		return
	}
	err := ldap2pg(ctx)
	if err != nil {
		if errs, ok := err.(interface{ Len() int }); ok {
			// Assume error are already logged before.
			slog.Error("Some errors occurred. See above for more details.", "err", err, "count", errs.Len())
		} else {
			slog.Error("Fatal error.", "err", err)
		}
		if internal.CurrentLevel > slog.LevelDebug {
			slog.Error("Run ldap2pg with --verbose to get more informations.")
		}
		os.Exit(1)
	}
}

func ldap2pg(ctx context.Context) (err error) {
	start := time.Now()

	stop, err := startProfiling()
	if err != nil {
		return
	}
	if stop != nil {
		defer stop()
	}
	defer postgres.CloseConn(ctx)

	controller, conf, err := configure()
	if err != nil {
		return
	}

	pc := conf.Postgres.Build()
	// Inspect session, running user, user options, blacklist, etc.
	instance, err := inspect.Stage0(ctx, pc)
	if err != nil {
		return
	}
	wantedRoles, wantedGrants, err := conf.Rules.Run(instance.RolesBlacklist, conf.Privileges)
	if err != nil {
		return
	}
	// Inspect users and databases (for drop owned by loop).
	err = instance.InspectStage1(ctx, pc)
	if err != nil {
		return
	}

	syncErrors := errorlist.New("synchronization errors")

	// Synchronize roles.
	queries := role.Diff(instance.AllRoles, instance.ManagedRoles, wantedRoles, instance.FallbackOwner, &instance.Databases)
	queries = postgres.GroupByDatabase(instance.Databases, instance.DefaultDatabase, queries)
	stageCount, err := postgres.Apply(ctx, queries, controller.Real)
	err = syncErrors.Extend(err)
	if err != nil {
		return
	}
	if 0 == stageCount {
		slog.Info("All roles synchronized.")
	}
	queryCount := stageCount

	// Synchronize privileges.
	if conf.ArePrivilegesManaged() {
		slog.Debug("Synchronizing privileges.")
		// Get the effective list of managed roles.
		managedRoles := mapset.NewSet(maps.Keys(wantedRoles)...)
		_, ok := instance.ManagedRoles["public"]
		if ok {
			managedRoles.Add("public")
		}

		instancePrivileges, objectPrivileges, defaultPrivileges := conf.Postgres.PrivilegesProfiles.BuildTypeMaps()

		// Start by default database. This allow to reuse the last
		// connexion openned when synchronizing roles.
		for _, dbname := range instance.Databases.SyncOrder(instance.DefaultDatabase, true) {
			slog.Debug("Stage 2: privileges.", "database", dbname)
			err := instance.InspectStage2(ctx, dbname, pc.SchemasQuery)
			if err != nil {
				return fmt.Errorf("inspect: %w", err)
			}
			var privs privileges.TypeMap
			if dbname == instance.DefaultDatabase {
				slog.Debug("Managing instance wide privileges.", "database", dbname)
				privs = make(privileges.TypeMap)
				maps.Copy(privs, instancePrivileges)
				maps.Copy(privs, objectPrivileges)
			} else {
				privs = objectPrivileges
			}
			stageCount, err := syncPrivileges(ctx, &controller, &instance, managedRoles, wantedGrants, dbname, privs)
			err = syncErrors.Extend(err)
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
			err = syncErrors.Extend(err)
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

	if syncErrors.Len() > 0 {
		return syncErrors
	}

	exitCode := controller.Finalize(
		start,
		len(wantedRoles),
		len(wantedGrants),
		queryCount,
	)
	os.Exit(exitCode)
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

func changeDirectory(directory string) (err error) {
	if directory == "" {
		return
	}
	slog.Debug("Changing directory.", "path", directory)
	return os.Chdir(directory)
}

// configure setup process settings from inputs
//
// Configures logging, environment, database connexion, etc.
func configure() (controller Controller, c config.Config, err error) {
	controller, err = unmarshalController()
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

	configPath := config.FindConfigFile(controller.Config)
	if configPath == "" {
		err = fmt.Errorf("no configuration file found")
		return
	}

	slog.Info("Using YAML configuration file.", "path", configPath)
	c, err = config.Load(configPath)
	if err != nil {
		return
	}

	if controller.SkipPrivileges {
		c.DropPrivileges()
	}

	envpath := config.FindDotEnvFile(configPath)
	if envpath != "" {
		slog.Debug("Loading .env file.", "path", envpath)
		err = godotenv.Load(envpath)
		if err != nil {
			err = fmt.Errorf(".env: %w", err)
			return
		}
	}

	if controller.Real {
		slog.Info("Real mode. Postgres instance will be modified.")
	} else {
		slog.Warn("Dry run. Postgres instance will be untouched.")
	}

	err = postgres.Configure(controller.Dsn)
	return
}

// syncPrivileges for a given database.
func syncPrivileges(ctx context.Context, controller *Controller, instance *inspect.Instance, roles mapset.Set[string], wantedGrants []privileges.Grant, dbname string, privs privileges.TypeMap) (int, error) {
	queryCount := 0
	allDatabases := maps.Keys(instance.Databases)
	acls := maps.Keys(privs)
	slices.Sort(acls)
	for _, acl := range acls {
		// synchronize ACL one at a time
		privs := privileges.TypeMap{acl: privs[acl]}
		expandedGrants := privileges.Expand(wantedGrants, privs, instance.Databases[dbname], allDatabases)
		currentGrants, err := instance.InspectGrants(ctx, dbname, privs, roles)
		// Special case, ignore grants on unmanaged databases.
		currentGrants = lists.Filter(currentGrants, func(g privileges.Grant) bool {
			if "DATABASE" != g.ACL() {
				return true
			}
			_, ok := instance.Databases[g.Object]
			return ok
		})

		if err != nil {
			return 0, fmt.Errorf("privileges: %w", err)
		}
		queries := privileges.Diff(currentGrants, expandedGrants)
		count, err := postgres.Apply(ctx, queries, controller.Real)
		if err != nil {
			return 0, fmt.Errorf("apply: %w", err)
		}
		slog.Debug("Privileges synchronized.", "acl", acl, "database", dbname)
		queryCount += count
	}
	return queryCount, nil
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
