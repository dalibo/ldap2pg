package wanted

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
)

func ExpandGrants(in []privilege.Grant, databases postgres.DBMap) (out []privilege.Grant) {
	for _, grant := range in {
		out = append(out, grant.Expand(databases)...)
	}
	return
}
