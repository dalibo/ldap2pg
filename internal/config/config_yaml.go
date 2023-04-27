package config

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"

	"github.com/dalibo/ldap2pg/internal/pyfmt"
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
func (config *Config) LoadYaml(root map[string]interface{}) (err error) {
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		_ = encoder.Encode(root)
		encoder.Close()
		slog.Debug("Normalized YAML:\n" + buf.String())
	}

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: decodeMapHook,
		Metadata:   &mapstructure.Metadata{},
		Result:     config,
	})
	if err != nil {
		return
	}
	err = d.Decode(root)
	if err != nil {
		return
	}

	for i := range config.SyncItems {
		config.SyncItems[i].InferAttributes()
	}

	slog.Debug("Loaded configuration file.", "version", config.Version)
	return
}

// Decode custom types for mapstructure. Implements mapstructure.DecodeHookFuncValue.
func decodeMapHook(from, to reflect.Value) (interface{}, error) {
	switch to.Type() {
	case reflect.TypeOf(pyfmt.Format{}):
		f := to.Interface().(pyfmt.Format)
		err := f.Parse(from.String())
		if err != nil {
			return nil, err
		}
		return f, nil
	case reflect.TypeOf(RoleOptions{}):
		r := to.Interface().(RoleOptions)
		r.LoadYaml(from.Interface().(map[string]interface{}))
		return r, nil
	}
	return from.Interface(), nil
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
