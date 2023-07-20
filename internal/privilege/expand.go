package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
	"golang.org/x/exp/slog"
)

type Expander interface {
	Expand(Grant, postgres.DBMap) []Grant
}

func Expand(in []Grant, databases postgres.DBMap) (out []Grant) {
	slog.Debug("Expanding wanted grants.")
	for _, grant := range in {
		if grant.IsDefault() {
			continue
		}

		e := Builtins[grant.Target]

		for _, g := range e.Expand(grant, databases) {
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

func ExpandDefault(in []Grant, databases postgres.DBMap) (out []Grant) {
	slog.Debug("Expanding wanted wanted privileges.")
	for _, grant := range in {
		var e Expander
		if !grant.IsDefault() {
			continue
		} else if "" == grant.Schema {
			e = Builtins["GLOBAL DEFAULT"]
		} else {
			e = Builtins["SCHEMA DEFAULT"]
		}

		for _, g := range e.Expand(grant, databases) {
			logAttrs := []interface{}{"grant", g}
			if "" != g.Database {
				logAttrs = append(logAttrs, "database", g.Database)
			}
			slog.Debug("Expand default privilege.", logAttrs...)
			out = append(out, g)
		}
	}
	return
}
