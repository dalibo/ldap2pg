package wanted

import (
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
)

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
			if results.Entry == nil || len(m.Name.Fields) == 0 {
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
