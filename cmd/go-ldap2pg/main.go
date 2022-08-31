package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"runtime/debug"

	. "github.com/dalibo/ldap2pg/internal/ldap2pg"
	"github.com/jackc/pgx/v4"
	"gopkg.in/yaml.v3"
)

var data string = `
toto: [1, "titi", null, 4.2]
`

type YamlConfig struct {
	Toto []interface{} `yaml:"toto"`
}

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

	err = LdapConnect(config)
	if err != nil {
		Logger.Fatal(err)
	}

	ctx := context.Background()
	pgconn, err := pgx.Connect(ctx, "")
	if err != nil {
		Logger.Fatalw("PostgreSQL connection error.", "error", err)
	}
	defer pgconn.Close(ctx)

	var me string
	err = pgconn.QueryRow(ctx, "SELECT CURRENT_USER;").Scan(&me)
	if err != nil {
		Logger.Fatalw("Failed to query PostgreSQL", "error", err)
	}

	Logger.Debugw("Introspected PostgreSQL user.", "username", me)

	y := YamlConfig{}
	err = yaml.Unmarshal([]byte(data), &y)
	if err != nil {
		Logger.Fatalw("Failed to parse YAML", "error", err)
	}
	log.Println("Len toto", len(y.Toto))
	for i, value := range y.Toto {
		switch t := value.(type) {
		case int:
			log.Printf("toto[%d] %T = %d", i, t, value.(int))
		case string:
			log.Printf("toto[%d] %T = %s", i, t, value.(string))
		default:
			log.Printf("toto[%d] %+v %T, unhandled.", i, value, t)
		}
	}
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
