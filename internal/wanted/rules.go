package wanted

import (
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
)

type GrantRule struct {
	Owner     pyfmt.Format
	Privilege pyfmt.Format
	Database  pyfmt.Format
	Schema    pyfmt.Format
	Object    pyfmt.Format
	To        pyfmt.Format `mapstructure:"role"`
}

func (r GrantRule) IsStatic() bool {
	return lists.And(r.Formats(), func(f pyfmt.Format) bool { return f.IsStatic() })
}

func (r GrantRule) Formats() []pyfmt.Format {
	return []pyfmt.Format{r.Owner, r.Privilege, r.Database, r.Schema, r.Object, r.To}
}

func (r GrantRule) Generate(results *ldap.Result, privs privileges.RefMap) <-chan privileges.Grant {
	ch := make(chan privileges.Grant)
	go func() {
		defer close(ch)
		if nil == results.Entry {
			alias := r.Privilege.Input
			for _, priv := range privs[alias] {
				// Case static rule.
				grant := privileges.Grant{
					Target:   priv.On,
					Grantee:  r.To.Input,
					Type:     priv.Type,
					Database: r.Database.Input,
					Schema:   r.Schema.Input,
					Object:   r.Object.Input,
				}
				if priv.IsDefault() {
					grant.Owner = r.Owner.Input
					grant.Object = ""
					if "global" == priv.Default {
						grant.Schema = ""
					} else if "__all__" == grant.Schema {
						// Use global default instead
						continue
					}
				}
				ch <- grant
			}
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Privilege, r.Database, r.Schema, r.Object, r.To) {
				alias := r.Privilege.Format(values)
				for _, priv := range privs[alias] {
					grant := privileges.Grant{
						Target:   priv.On,
						Grantee:  r.To.Format(values),
						Type:     priv.Type,
						Database: r.Database.Format(values),
						Schema:   r.Schema.Format(values),
						Object:   r.Object.Format(values),
					}
					if priv.IsDefault() {
						grant.Owner = r.Owner.Input
						grant.Object = ""
						if "global" == priv.Default {
							grant.Schema = ""
						} else if "__all__" == grant.Schema {
							// Use global default instead
							continue
						}
					}
					ch <- grant
				}
			}
		}
	}()
	return ch
}

type RoleRule struct {
	Name         pyfmt.Format
	Options      role.Options
	Comment      pyfmt.Format
	Parents      []MembershipRule
	Config       *role.Config
	BeforeCreate pyfmt.Format `mapstructure:"before_create"`
	AfterCreate  pyfmt.Format `mapstructure:"after_create"`
}

func (r RoleRule) IsStatic() bool {
	return lists.And(r.Formats(), func(f pyfmt.Format) bool { return f.IsStatic() })
}

func (r RoleRule) Formats() []pyfmt.Format {
	fmts := []pyfmt.Format{r.Name, r.Comment, r.BeforeCreate, r.AfterCreate}
	for _, p := range r.Parents {
		fmts = append(fmts, p.Name)
	}
	return fmts
}

func (r RoleRule) Generate(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
	go func() {
		defer close(ch)
		parents := []role.Membership{}
		for _, m := range r.Parents {
			if nil == results.Entry || 0 == len(m.Name.Fields) {
				// Static case.
				parents = append(parents, m.Generate(nil))
			} else {
				// Dynamic case.
				for values := range results.GenerateValues(m.Name) {
					parents = append(parents, m.Generate(values))
				}
			}
		}

		if nil == results.Entry {
			// Case static rule.
			role := role.Role{
				Name:         r.Name.String(),
				Comment:      r.Comment.String(),
				Options:      r.Options,
				Parents:      parents,
				Config:       r.Config,
				BeforeCreate: r.BeforeCreate.String(),
				AfterCreate:  r.AfterCreate.String(),
			}
			ch <- role
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Name, r.Comment, r.BeforeCreate, r.AfterCreate) {
				role := role.Role{}
				role.Name = r.Name.Format(values)
				role.Comment = r.Comment.Format(values)
				role.Options = r.Options
				role.Parents = append(parents[0:0], parents...) // copy
				role.BeforeCreate = r.BeforeCreate.Format(values)
				role.AfterCreate = r.AfterCreate.Format(values)
				ch <- role
			}
		}
	}()
	return ch
}

type MembershipRule struct {
	Name pyfmt.Format
}

func (m MembershipRule) String() string {
	return m.Name.String()
}

func (m MembershipRule) IsStatic() bool {
	return m.Name.IsStatic()
}

func (m MembershipRule) Generate(values map[string]string) role.Membership {
	return role.Membership{
		Name: m.Name.Format(values),
	}
}
