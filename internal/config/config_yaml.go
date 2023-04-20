package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// Marshall YAML from file path or stdin if path is -.
func ReadYaml(path string) (values interface{}, err error) {
	var fo io.ReadCloser
	if path == "-" {
		slog.Info("Reading configuration from standard input.")
		fo = os.Stdin
	} else {
		fo, err = os.Open(path)
		if err != nil {
			return
		}
	}
	defer fo.Close()
	dec := yaml.NewDecoder(fo)
	err = dec.Decode(&values)
	return
}

// Fill configuration from YAML data.
func (config *Config) LoadYaml(yamlData interface{}) (err error) {
	root, err := NormalizeConfigRoot(yamlData)
	if err != nil {
		return
	}

	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		_ = encoder.Encode(root)
		encoder.Close()
		slog.Debug("Normalized YAML:\n" + buf.String())
	}

	err = config.LoadVersion(root)
	if err != nil {
		return
	}
	if config.Version != 5 {
		err = errors.New("Unsupported configuration version")
		return
	}

	postgres, found := root["postgres"]
	if found {
		err = config.LoadYamlPostgres(postgres)
		if err != nil {
			return
		}
	}

	syncMap := root["sync_map"]
	err = config.LoadYamlSyncMap(syncMap.([]interface{}))
	return
}

func (config *Config) LoadVersion(yaml map[string]interface{}) (err error) {
	version, ok := yaml["version"]
	if !ok {
		version = 5
	}
	config.Version, ok = version.(int)
	if !ok {
		err = errors.New("Configuration version must be integer")
		return
	}
	return
}

func (config *Config) LoadYamlPostgres(postgres interface{}) (err error) {
	var postgresMap map[string]interface{}

	switch t := postgres.(type) {
	case map[string]interface{}:
		postgresMap = postgres.(map[string]interface{})
	case nil:
		err = fmt.Errorf("postgres: section must not be null")
		return
	default:
		err = fmt.Errorf("postgres: section must be a map, got %v (%T)", postgres, t)
		return
	}

	knownQueries := []*InspectQuery{
		&config.Postgres.DatabasesQuery,
		&config.Postgres.ManagedRolesQuery,
		&config.Postgres.RolesBlacklistQuery,
	}

	for _, q := range knownQueries {
		value, ok := postgresMap[q.Name]
		if !ok {
			continue
		}
		slog.Debug("Loading Postgres query from YAML.",
			"query", q.Name)

		q.Value = value
	}
	return
}

func (config *Config) LoadYamlSyncMap(yaml []interface{}) (err error) {
	for _, iItem := range yaml {
		var item SyncItem
		err = item.LoadYaml(iItem.(map[string]interface{}))
		if err != nil {
			return
		}
		config.SyncMap = append(config.SyncMap, item)
	}
	return
}
