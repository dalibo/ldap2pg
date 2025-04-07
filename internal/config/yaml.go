package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/jackc/pgx/v5"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

// Marshall YAML from file path or stdin if path is -.
func ReadYaml(path string) (values any, err error) {
	var fo io.ReadCloser
	if path == "<stdin>" {
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
func (c *Config) LoadYaml(root map[string]any) (err error) {
	err = c.DecodeYaml(root)
	if err != nil {
		return
	}

	for i := range c.Rules {
		item := &c.Rules[i]
		item.InferAttributes()
		// states.ComputeWanted is simplified base on the assumption
		// there is no more than one sub-search. Fail otherwise.
		if 1 < len(item.LdapSearch.Subsearches) {
			err = fmt.Errorf("multiple sub-search unsupported")
			return
		}
		item.ReplaceAttributeAsSubentryField()
	}

	slog.Debug("Loaded configuration file.", "version", c.Version)
	return
}

func Dump(root any) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	_ = encoder.Encode(root)
	encoder.Close()
	color := isatty.IsTerminal(os.Stderr.Fd())
	slog.Debug("Dumping normalized YAML to stderr.")
	if color {
		os.Stderr.WriteString("\033[0;2m")
	}
	os.Stderr.WriteString(buf.String())
	if color {
		os.Stderr.WriteString("\033[0m")
	}
}

// Wrap mapstructure for config object
func (c *Config) DecodeYaml(yaml any) (err error) {
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       decodeMapHook,
		Metadata:         &mapstructure.Metadata{},
		Result:           c,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return
	}
	err = d.Decode(yaml)
	return
}

// Decode custom types for mapstructure. Implements mapstructure.DecodeHookFuncValue.
func decodeMapHook(from, to reflect.Value) (any, error) {
	switch to.Type() {
	case reflect.TypeOf(pyfmt.Format{}):
		f := to.Interface().(pyfmt.Format)
		err := f.Parse(from.String())
		if err != nil {
			return nil, err
		}
		return f, nil
	case reflect.TypeOf(QueryConfig[string]{}):
		v := to.Interface().(QueryConfig[string])
		v.Value = from.Interface()
		err := v.Instantiate(pgx.RowTo[string], YamlTo[string])
		if err != nil {
			return nil, err
		}
		return v, nil
	case reflect.TypeOf(QueryConfig[postgres.Schema]{}):
		v := to.Interface().(QueryConfig[postgres.Schema])
		v.Value = from.Interface()
		err := v.Instantiate(postgres.RowToSchema, postgres.YamlToSchema)
		if err != nil {
			return nil, err
		}
		return v, nil
	case reflect.TypeOf(ldap.Scope(1)):
		s, err := ldap.ParseScope(from.String())
		if err != nil {
			return from.Interface(), err
		}
		return s, nil
	}
	return from.Interface(), nil
}

func (c *Config) checkVersion(yaml any) (err error) {
	yamlMap, ok := yaml.(map[string]any)
	if !ok {
		return errors.New("YAML is not a map")
	}
	version, ok := yamlMap["version"]
	if !ok {
		slog.Debug("Fallback to version 5.")
		version = 5
	}
	c.Version, ok = version.(int)
	if !ok {
		return errors.New("configuration version must be integer")
	}
	if c.Version != 6 {
		slog.Debug("Unsupported configuration version. Minimum version is 6.", "version", c.Version)
		return errors.New("configuration version must be 6")
	}
	return
}
