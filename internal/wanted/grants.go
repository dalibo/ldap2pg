package wanted

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"golang.org/x/exp/slog"
)

type GrantExpander interface {
	Expand(privilege.Grant, postgres.DBMap) []privilege.Grant
}

var expanders map[string]GrantExpander

func init() {
	expanders = make(map[string]GrantExpander)
	for k, p := range privilege.Builtins {
		var i GrantExpander

		if "GLOBAL DEFAULT" == p.Object {
			i = privilege.NewGlobalDefault(p.Object, p.Inspect)
		} else if "SCHEMA DEFAULT" == p.Object {
			i = privilege.NewSchemaDefault(p.Object, p.Inspect)
		} else if strings.HasPrefix(p.Object, "ALL ") {
			i = privilege.NewAll(p.Object, p.Inspect)
		} else if "instance" == p.Scope {
			i = privilege.NewInstance(p.Object, p.Inspect)
		} else if "database" == p.Scope {
			i = privilege.NewDatabase(p.Object, p.Inspect)
		} else {
			panic("unsupported")
		}
		expanders[k] = i
	}
}

func ExpandGrants(in []privilege.Grant, databases postgres.DBMap, rolesBlacklist lists.Blacklist) (out []privilege.Grant) {
	slog.Debug("Expanding wanted grants.")
	for _, grant := range in {
		var e GrantExpander
		if !grant.IsDefault() {
			e = expanders[grant.Target]
		} else if "" == grant.Schema {
			e = expanders["GLOBAL DEFAULT"]
		} else {
			e = expanders["SCHEMA DEFAULT"]
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
