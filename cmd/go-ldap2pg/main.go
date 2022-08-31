package main

import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"

	. "github.com/dalibo/ldap2pg/internal/ldap2pg"
)

func main() {
	err := SetupLogging()
	if err != nil {
		log.Panicf("Failed to setup logging: %s", err)
	}
	defer Logger.Sync() //nolint:errcheck

	config, err := LoadConfig()
	if err != nil {
		Logger.Panicw("Failed to load configuration.", "error", err)
	}
	switch config.Action {
	case ShowHelpAction:
		ShowHelp()
		return
	case ShowVersionAction:
		showVersion()
		return
	case RunAction:
	}

	LogLevel.SetLevel(config.LogLevel)
	Logger.Infow("Starting ldap2pg", "commit", ShortRevision, "version", Version, "runtime", runtime.Version())
	Logger.Infow("Using YAML configuration file.", "path", config.ConfigFile)

	err = LdapConnect(config)
	if err != nil {
		Logger.Fatal(err)
	}

	err = PostgresConnect(config)
	if err != nil {
		Logger.Fatal(err)
	}

	Logger.Info("Doing nothing yet.")
}

func showVersion() {
	fmt.Printf("go-ldap2pg %s\n", Version)

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	modmap := make(map[string]string)
	for _, mod := range bi.Deps {
		modmap[mod.Path] = mod.Version
	}
	modules := []string{
		"github.com/jackc/pgx/v4",
		"github.com/go-ldap/ldap/v3",
		"gopkg.in/yaml.v3",
	}
	for _, mod := range modules {
		fmt.Printf("%s %s\n", mod, modmap[mod])
	}

	fmt.Printf("%s %s %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
