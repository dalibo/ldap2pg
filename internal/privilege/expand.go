package privilege

import (
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"golang.org/x/exp/slog"
)

type Expander interface {
	Expand(Grant, postgres.DBMap) []Grant
}

func Expand(in []Grant, databases postgres.DBMap, rolesBlacklist lists.Blacklist) (out []Grant) {
	slog.Debug("Expanding wanted grants.")
	for _, grant := range in {
		var e Expander
		if !grant.IsDefault() {
			e = Builtins[grant.Target]
		} else if "" == grant.Schema {
			e = Builtins["GLOBAL DEFAULT"]
		} else {
			e = Builtins["SCHEMA DEFAULT"]
		}

		for _, g := range e.Expand(grant, databases) {
			pattern := rolesBlacklist.MatchString(g.Owner)
			if "" != pattern {
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
