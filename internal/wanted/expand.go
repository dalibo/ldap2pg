package wanted

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/slog"
)

// ExpandGrants wanted grants.
func ExpandGrants(grants []privilege.Grant, databases []postgres.Database) (out []privilege.Grant) {
	for _, grant := range grants {
		p := grant.Privilege()
		for _, expansion := range p.Expand(grant, databases) {
			expansion.Normalize()
			slog.Debug("Wants grant.", "grant", expansion)
			out = append(out, expansion)
		}
	}
	return
}
