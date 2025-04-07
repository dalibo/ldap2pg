package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"slices"
)

var (
	commit   string
	versions = make(map[string]string)
	mainDeps = []string{
		"github.com/jackc/pgx/v5",
		"github.com/go-ldap/ldap/v3",
		"gopkg.in/yaml.v3",
	}
)

func version() string {
	return versions["github.com/dalibo/ldap2pg"]
}

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("Failed to read build information.")
	}
	for _, mod := range bi.Deps {
		if slices.Contains(mainDeps, mod.Path) {
			versions[mod.Path] = mod.Version
		}
		if len(versions) >= len(mainDeps) {
			break
		}
	}

	versions[bi.Main.Path] = bi.Main.Version

	for i := range bi.Settings {
		if bi.Settings[i].Key == "vcs.revision" {
			commit = bi.Settings[i].Value
			break
		}
	}
}

func showVersion() {
	fmt.Printf("ldap2pg %s\n", version())

	for _, path := range mainDeps {
		fmt.Printf("%s %s\n", path, versions[path])
	}

	fmt.Printf("%s %s %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
