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
	"github.com/knadh/koanf/v2"
)

// Avoid error if file does not exist.
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

func (looseFileProvider) Read() (map[string]any, error) {
	panic("not implemented")
}

// parser returns ldaprc as plain map for koanf.
// delim defines the nesting hierarchy of keys.
type parser struct {
	delim string
}

func (p parser) Unmarshal(data []byte) (map[string]any, error) {
	out := make(map[string]any)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	re := regexp.MustCompile(`\s+`)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(line)
		if "" == line {
			continue
		}
		fields := re.Split(line, 2)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		out[fields[0]] = fields[1]
	}
	return maps.Unflatten(out, p.delim), nil
}

func (parser) Marshal(map[string]any) ([]byte, error) {
	panic("not implemented")
}
