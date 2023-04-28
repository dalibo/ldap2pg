// Implements ldap.conf(5)
package ldap

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/exp/slog"
)

var knownOptions = []string{
	"BASE",
	"BINDDN",
	"PASSWORD", // ldap2pg extension.
	"REFERRALS",
	"TIMEOUT",
	"TLS_REQCERT",
	"NETWORK_TIMEOUT",
	"URI",
}

// Holds options with their raw value from either file of env. Marshaling is
// done on demand by getters.
type OptionsMap map[string]RawOption

type RawOption struct {
	Key    string
	Value  string
	Origin string
}

func Initialize() (options OptionsMap, err error) {
	_, ok := os.LookupEnv("LDAPNOINIT")
	if ok {
		slog.Debug("Skip LDAP initialization.")
		return
	}
	path := "/etc/ldap/ldap.conf"
	home, _ := os.UserHomeDir()
	options = make(OptionsMap)
	options.LoadDefaults()
	err = options.LoadFiles(
		path,
		filepath.Join(home, "ldaprc"),
		filepath.Join(home, ".ldaprc"),
		"ldaprc",
	)
	if err != nil {
		return
	}
	path = os.Getenv("LDAPCONF")
	if "" != path {
		err = options.LoadFiles(path)
		if err != nil {
			return
		}
	}
	path = os.Getenv("LDAPRC")
	if "" != path {
		err = options.LoadFiles(
			filepath.Join(home, path),
			fmt.Sprintf("%s/.%s", home, path),
			"./"+path,
		)
	}
	options.LoadEnv()
	return
}

func (m OptionsMap) GetString(name string) string {
	option, ok := m[name]
	if ok {
		slog.Debug("Read LDAP option.", "key", option.Key, "origin", option.Origin)
		return option.Value
	}
	return ""
}

func (m *OptionsMap) LoadDefaults() {
	defaults := map[string]string{
		"TLS_REQCERT": "try",
	}
	for key, value := range defaults {
		(*m)[key] = RawOption{
			Key:    key,
			Value:  value,
			Origin: "default",
		}
	}
}

func (m *OptionsMap) LoadEnv() {
	for _, name := range knownOptions {
		envName := "LDAP" + name
		value, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}
		option := RawOption{
			Key:    strings.TrimPrefix(envName, "LDAP"),
			Value:  value,
			Origin: "env",
		}
		(*m)[option.Key] = option
	}
}

func (m *OptionsMap) LoadFiles(path ...string) (err error) {
	for _, candidate := range path {
		if !filepath.IsAbs(candidate) {
			candidate, _ = filepath.Abs(candidate)
		}
		_, err := os.Stat(candidate)
		if err != nil {
			slog.Debug("Ignoring configuration file.", "path", candidate, "err", err.Error())
			continue
		}
		slog.Debug("Found LDAP configuration file.", "path", candidate)
		for item := range iterFileOptions(candidate) {
			err, _ := item.(error)
			if err != nil {
				return err
			}
			option := item.(RawOption)
			(*m)[option.Key] = option
		}
	}
	return
}

func iterFileOptions(path string) <-chan any {
	ch := make(chan any)
	fo, err := os.Open(path)
	if err != nil {
		defer close(ch)
		ch <- err
	} else {
		go func() {
			defer close(ch)
			scanner := bufio.NewScanner(fo)
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
				var item RawOption
				item.Key = strings.ToUpper(fields[0])
				item.Value = fields[1]
				item.Origin = path
				ch <- item
			}
		}()
	}
	return ch
}
