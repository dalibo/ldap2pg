package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
)

type Expander interface {
	Expand(Grant, postgres.Database, []string) []Grant
}

func Expand(in []Grant, privileges TypeMap, database postgres.Database, databases []string) (out []Grant) {
	for _, grant := range in {
		k := grant.PrivilegeKey()
		_, ok := privileges[k]
		if !ok {
			continue
		}

		e := Builtins[k]
		out = append(out, e.Expand(grant, database, databases)...)
	}
	return
}
