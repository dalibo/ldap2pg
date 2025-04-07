package ldap

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

var KnownRDNs = []string{"cn", "l", "st", "o", "ou", "c", "street", "dc", "uid"}

type Config struct {
	KnownRDNs []string `mapstructure:"known_rdns"`
}

func (c Config) apply() {
	if c.KnownRDNs != nil {
		slog.Debug("Setting known RDNs.", "known_rdns", c.KnownRDNs)
		KnownRDNs = c.KnownRDNs
	}
}

var k = koanf.New(".")

// cf. https://git.openldap.org/openldap/openldap/-/blob/bf01750381726db3052d94514eec4048c90a616a/libraries/libldap/init.c#L640
func Initialize(conf Config) error {
	conf.apply()
	_, ok := os.LookupEnv("LDAPNOINIT")
	if ok {
		slog.Debug("Skip LDAP initialization.")
		return nil
	}

	_ = k.Load(confmap.Provider(map[string]any{
		"URI":             "ldap://localhost",
		"NETWORK_TIMEOUT": "30",
		"RC":              "ldaprc",
		"TLS_REQCERT":     "try",
		"TIMEOUT":         "30",
	}, k.Delim()), nil)

	_ = k.Load(env.Provider("LDAP", k.Delim(), func(key string) string {
		slog.Debug("Loading LDAP environment var.", "var", key)
		return strings.TrimPrefix(key, "LDAP")
	}), nil)

	_ = k.Load(posflag.ProviderWithFlag(pflag.CommandLine, k.Delim(), k, func(f *pflag.Flag) (string, any) {
		if !strings.HasPrefix(f.Name, "ldap") {
			return "", nil
		}
		// Rename LDAP flags
		// e.g. --ldapppassword_file -> PASSWORD_FILE
		key := strings.ToUpper(f.Name)
		key = strings.TrimPrefix(key, "LDAP")
		key = strings.ReplaceAll(key, "-", "_")
		return key, posflag.FlagVal(pflag.CommandLine, f)
	}), nil)

	passwordFilePath := k.String("PASSWORD_FILE")
	if passwordFilePath != "" {
		slog.Debug("Reading password from file.", "path", passwordFilePath)
		data, err := readSecretFromFile(passwordFilePath)
		if err != nil {
			return fmt.Errorf("ldap password: %w", err)
		}
		// Set() only throws error when using StrictMerge which is not the case.
		_ = k.Set("PASSWORD", data)
	}

	// cf. https://git.openldap.org/openldap/openldap/-/blob/bf01750381726db3052d94514eec4048c90a616a/libraries/libldap/init.c#L741
	home, _ := os.UserHomeDir()
	files := []string{
		"/etc/ldap/ldap.conf",
		filepath.Join(home, "ldaprc"),
		filepath.Join(home, ".ldaprc"),
		"ldaprc", // search in CWD
		// Read CONF and RC only from env, before above files are effectively read.
		k.String("CONF"),
		filepath.Join(home, k.String("RC")),
		filepath.Join(home, fmt.Sprintf(".%s", k.String("RC"))),
		k.String("RC"), // Search in CWD.
	}
	for _, candidate := range files {
		if candidate == "" {
			continue
		}

		err := k.Load(newLooseFileProvider(candidate), parser{k.Delim()})
		if err != nil {
			return fmt.Errorf("%s: %w", candidate, err)
		}
	}
	return nil
}

// readSecretFromFile reads a file and returns its content.
// It returns an error if the file does not exist or has too open permissions.
func readSecretFromFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if (info.Mode().Perm() & 0o007) != 0 {
		return "", errors.New("permissions too wide")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
