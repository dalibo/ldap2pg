package wanted

import (
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
)

type GrantRule struct {
	Privilege pyfmt.Format
	Database  pyfmt.Format
	Schema    pyfmt.Format
	Object    pyfmt.Format
	To        pyfmt.Format
}

func (r GrantRule) IsStatic() bool {
	return r.Privilege.IsStatic() &&
		r.Database.IsStatic() &&
		r.Schema.IsStatic() &&
		r.Object.IsStatic() &&
		r.To.IsStatic()
}

func (r GrantRule) Generate(results *ldap.Result, privileges privilege.RefMap) <-chan privilege.Grant {
	ch := make(chan privilege.Grant)
	go func() {
		defer close(ch)
		if nil == results.Entry {
			alias := r.Privilege.Input
			for _, priv := range privileges[alias] {
				// Case static rule.
				grant := privilege.Grant{
					Target:   priv.On,
					Grantee:  r.To.Input,
					Type:     priv.Type,
					Database: r.Database.Input,
					Schema:   r.Schema.Input,
					Object:   r.Object.Input,
				}
				ch <- grant

			}
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Privilege, r.Database, r.Schema, r.Object, r.To) {
				alias := r.Privilege.Format(values)
				for _, priv := range privileges[alias] {
					grant := privilege.Grant{
						Target:   priv.On,
						Grantee:  r.To.Format(values),
						Type:     priv.Type,
						Database: r.Database.Format(values),
						Schema:   r.Schema.Format(values),
						Object:   r.Object.Format(values),
					}
					ch <- grant
				}
			}
		}
	}()
	return ch
}

type RoleRule struct {
	Name    pyfmt.Format
	Options role.Options
	Comment pyfmt.Format
	Parents []pyfmt.Format
}

func (r RoleRule) IsStatic() bool {
	return r.Name.IsStatic() &&
		r.Comment.IsStatic() &&
		lists.And(r.Parents, func(f pyfmt.Format) bool { return f.IsStatic() })
}

func (r RoleRule) Generate(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
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
			// Case static rule.
			role := role.Role{
				Name:    r.Name.String(),
				Comment: r.Comment.String(),
				Options: r.Options,
				Parents: parents,
			}
			ch <- role
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Name, r.Comment) {
				role := role.Role{}
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
