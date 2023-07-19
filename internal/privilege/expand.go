package privilege

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"golang.org/x/exp/slog"
)

type Expander interface {
	Expand(Grant, postgres.DBMap) []Grant
}

var expanders map[string]Expander

func init() {
	expanders = make(map[string]Expander)
	for k, p := range Builtins {
		var i Expander

		if "GLOBAL DEFAULT" == p.Object {
			i = NewGlobalDefault(p.Object, p.Inspect)
		} else if "SCHEMA DEFAULT" == p.Object {
			i = NewSchemaDefault(p.Object, p.Inspect)
		} else if strings.HasPrefix(p.Object, "ALL ") {
			i = NewAll(p.Object, p.Inspect)
		} else if "instance" == p.Scope {
			i = NewInstance(p.Object, p.Inspect)
		} else if "database" == p.Scope {
			i = NewDatabase(p.Object, p.Inspect)
		} else {
			panic("unsupported")
		}
		expanders[k] = i
	}
}

func Expand(in []Grant, databases postgres.DBMap, rolesBlacklist lists.Blacklist) (out []Grant) {
	slog.Debug("Expanding wanted grants.")
	for _, grant := range in {
		var e Expander
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
