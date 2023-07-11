package wanted

import (
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/slog"
)

func ExpandGrants(in []privilege.Grant, databases postgres.DBMap, rolesBlacklist lists.Blacklist) (out []privilege.Grant) {
	for _, grant := range in {
		for _, g := range grant.Expand(databases) {
			pattern := rolesBlacklist.MatchString(g.Owner)
			if "" != pattern {
				slog.Debug("Ignoring blacklisted owner.", "grant", g, "pattern", pattern)
				continue
			}
			out = append(out, g)
		}
	}
	return
}
