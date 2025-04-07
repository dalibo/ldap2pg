package role

import "strings"

type Config map[string]string

func (c Config) Parse(rows []string) {
	for _, row := range rows {
		parts := strings.SplitN(row, "=", 2)
		if len(parts) != 2 {
			continue
		}
		c[parts[0]] = parts[1]
	}
}
