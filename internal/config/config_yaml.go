package config

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// Implements config.YamlToFunc. Similar to pgx.RowTo.
func YamlToString(value interface{}) (pattern string, err error) {
	pattern = value.(string)
	return
}

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
	err = config.checkVersion(yamlData)
	if err != nil {
		return
	}
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

	err = mapstructure.Decode(root, config)
	slog.Debug("Loaded configuration file.", "version", config.Version)
	return
}

func (config *Config) checkVersion(yaml interface{}) (err error) {
	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		return errors.New("YAML is not a map")
	}
	version, ok := yamlMap["version"]
	if !ok {
		slog.Debug("Fallback to version 5.")
		version = 5
	}
	config.Version, ok = version.(int)
	if !ok {
		return errors.New("Configuration version must be integer")
	}
	if config.Version != 5 {
		slog.Debug("Unsupported configuration version.", "version", config.Version)
		return errors.New("Unsupported configuration version")
	}
	return
}
