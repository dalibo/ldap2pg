package sync

import (
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
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
