package role

type Membership struct {
	Grantor string
	Name    string
}

func (m Membership) String() string {
	return m.Name
}

func (r Role) MemberOf(p string) bool {
	for _, m := range r.Parents {
		if p == m.Name {
			return true
		}
	}
	return false
}

func (r Role) MissingParents(o []Membership) (out []Membership) {
	for _, m := range o {
		if !r.MemberOf(m.Name) {
			out = append(out, m)
		}
	}
	return
}
