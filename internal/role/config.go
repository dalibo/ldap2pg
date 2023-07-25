package role

import "strings"

type Config map[string]string

func (c Config) Parse(rows []string) {
	for _, row := range rows {
		parts := strings.SplitN(row, "=", 2)
		if 2 != len(parts) {
			continue
		}
		c[parts[0]] = parts[1]
	}
}
