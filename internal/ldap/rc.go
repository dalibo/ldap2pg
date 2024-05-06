// Implements ldap.conf(5)
package ldap

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New("_")

// cf. https://git.openldap.org/openldap/openldap/-/blob/bf01750381726db3052d94514eec4048c90a616a/libraries/libldap/init.c#L640
func Initialize() error {
	_, ok := os.LookupEnv("LDAPNOINIT")
	if ok {
		slog.Debug("Skip LDAP initialization.")
		return nil
	}

	_ = k.Load(confmap.Provider(map[string]interface{}{
		"URI":             "ldap://localhost",
		"NETWORK_TIMEOUT": "30",
		"RC":              "ldaprc",
		"TLS_REQCERT":     "try",
		"TIMEOUT":         "30",
	}, "_"), nil)

	_ = k.Load(env.Provider("LDAP", "_", func(key string) string {
		return strings.TrimPrefix(key, "LDAP")
	}), nil)

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

		err := k.Load(newLooseFileProvider(candidate), parser{})
		if err != nil {
			return fmt.Errorf("%s: %w", candidate, err)
		}
	}
	return nil
}

// looseFileProvider reads a file if it exists.
type looseFileProvider struct {
	path string
}

func newLooseFileProvider(path string) koanf.Provider {
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(path)
	}
	return looseFileProvider{path: path}
}

func (p looseFileProvider) ReadBytes() ([]byte, error) {
	data, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	slog.Debug("Found LDAP configuration file.", "path", p.path, "err", err)
	return data, err
}

func (looseFileProvider) Read() (map[string]interface{}, error) {
	panic("not implemented")
}

// parser returns ldaprc as plain map for koanf.
type parser struct{}

func (parser) Unmarshal(data []byte) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	re := regexp.MustCompile(`\s+`)
	for scanner.Scan() {
		line := scanner.Text()
		slog.Debug("LDAP config line.", "line", line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(line)
		if "" == line {
			continue
		}
		fields := re.Split(line, 2)
		slog.Debug("LDAP config line.", "key", fields[0], "value", fields[1])
		out[fields[0]] = fields[1]
	}
	return maps.Unflatten(out, "_"), nil
}

func (parser) Marshal(map[string]interface{}) ([]byte, error) {
	panic("not implemented")
}
