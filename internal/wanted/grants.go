package wanted

import (
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/slog"
)

func ExpandGrants(in []privilege.Grant, databases postgres.DBMap, rolesBlacklist lists.Blacklist) (out []privilege.Grant) {
	for _, grant := range in {
		p := grant.Privilege()
		for _, g := range p.Expand(grant, databases) {
			pattern := rolesBlacklist.MatchString(g.Owner)
			if "" != pattern {
				slog.Debug("Ignoring blacklisted owner.", "grant", g, "pattern", pattern)
				continue
			}
			logAttrs := []interface{}{"grant", g}
			if "" != g.Database {
				logAttrs = append(logAttrs, "database", g.Database)
			}
			slog.Debug("Expand grant.", logAttrs...)
			out = append(out, g)
		}
	}
	return
}
