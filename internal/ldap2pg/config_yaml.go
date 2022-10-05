package ldap2pg

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Marshall YAML from file path or stdin if path is -.
func ReadYaml(path string) (values interface{}, err error) {
	var fo io.ReadCloser
	if path == "-" {
		log.Info("Reading configuration from standard input.")
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
func (config *Config) LoadYaml(values interface{}) (err error) {
	yamlMap, err := ensureYamlMap(values)
	if err != nil {
		return
	}

	err = config.checkVersion(yamlMap)
	if err != nil {
		return
	}

	postgres, found := yamlMap["postgres"]
	if found {
		err = config.loadYamlPostgres(postgres)
	}
	return
}

func (config *Config) checkVersion(yamlMap map[string]interface{}) (err error) {
	version, ok := yamlMap["version"]
	if !ok {
		version = 5
	}

	switch version.(type) {
	case int:
		if version != 5 {
			err = fmt.Errorf("Unsupported configuration version %v", version)
		} else {
			config.Version = version.(int)
		}

	default:
		err = fmt.Errorf("Bad version number: %v", version)
	}
	return
}

func ensureYamlMap(values interface{}) (yamlMap map[string]interface{}, err error) {
	switch t := values.(type) {
	case map[string]interface{}:
		yamlMap = values.(map[string]interface{})
	case []interface{}:
		yamlMap = make(map[string]interface{})
		yamlMap["sync_map"] = values.([]interface{})
	case nil:
		err = fmt.Errorf("YAML is empty")
		return
	default:
		err = fmt.Errorf("Bad YAML document root: %v (%T)", values, t)
		return
	}
	return
}

func (config *Config) loadYamlPostgres(postgres interface{}) (err error) {
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

	knownQueries := []*Query{
		&config.Postgres.DatabasesQuery,
		&config.Postgres.ManagedRolesQuery,
		&config.Postgres.RolesBlacklistQuery,
	}

	for _, q := range knownQueries {
		value, ok := postgresMap[q.Name]
		if !ok {
			continue
		}
		log.
			WithField("query", q.Name).
			Debug("Loading Postgres query from YAML.")
		q.Value = value
	}
	return
}
