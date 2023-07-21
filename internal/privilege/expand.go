package privilege

import (
	"github.com/dalibo/ldap2pg/internal/postgres"
)

type Expander interface {
	Expand(Grant, postgres.DBMap) []Grant
}

func Expand(in []Grant, databases postgres.DBMap) (out []Grant) {
	for _, grant := range in {
		if grant.IsDefault() {
			continue
		}

		e := Builtins[grant.Target]
		out = append(out, e.Expand(grant, databases)...)
	}
	return
}

func ExpandDefault(in []Grant, databases postgres.DBMap) (out []Grant) {
	for _, grant := range in {
		var e Expander
		if !grant.IsDefault() {
			continue
		} else if "" == grant.Schema {
			e = Builtins["GLOBAL DEFAULT"]
		} else {
			e = Builtins["SCHEMA DEFAULT"]
		}

		out = append(out, e.Expand(grant, databases)...)
	}
	return
}
