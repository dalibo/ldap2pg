package search

import (
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/roles"
)

type RoleRule struct {
	Name    pyfmt.Format
	Options roles.Options
	Comment pyfmt.Format
	Parents []pyfmt.Format
}

func (r RoleRule) IsStatic() bool {
	if 0 < len(r.Name.Fields) {
		return false
	}
	if 0 < len(r.Comment.Fields) {
		return false
	}
	for _, f := range r.Parents {
		if 0 < len(f.Fields) {
			return false
		}
	}
	return true
}
