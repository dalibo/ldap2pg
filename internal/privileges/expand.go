package privileges

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
)

type expander interface {
	Expand(Grant, postgres.Database, []string) []Grant
}

// Expand grants from rules.
//
// e.g.: instantiate a grant on all databases for each database.
// Same for schemas.
func Expand(in []Grant, privileges TypeMap, database postgres.Database, databases []string) (out []Grant) {
	for _, grant := range in {
		k := grant.ACL()
		_, ok := privileges[k]
		if !ok {
			continue
		}

		e := ACLs[k]
		out = append(out, e.Expand(grant, database, databases)...)
	}
	return
}
