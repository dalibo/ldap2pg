package sync

import (
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/roles"
	mapset "github.com/deckarep/golang-set/v2"
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

func (r RoleRule) Generate(results *ldap.Results) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		parents := mapset.NewSet[string]()
		for _, f := range r.Parents {
			if nil == results.Entry || 0 == len(f.Fields) {
				// Static case.
				parents.Add(f.String())
			} else {
				// Dynamic case.
				for values := range results.GenerateValues(f) {
					parents.Add(f.Format(values))
				}
			}
		}

		if nil == results.Entry {
			// Case static role.
			role := roles.Role{}
			role.Name = r.Name.String()
			role.Comment = r.Comment.String()
			role.Options = r.Options
			role.Parents = parents
			ch <- role
		} else {
			// Case dynamic roles.
			for values := range results.GenerateValues(r.Name, r.Comment) {
				role := roles.Role{}
				role.Name = r.Name.Format(values)
				role.Comment = r.Comment.Format(values)
				role.Options = r.Options
				role.Parents = parents.Clone()
				ch <- role
			}
		}
	}()
	return ch
}
