package ldap

import "golang.org/x/exp/maps"

type Search struct {
	Base        string
	Scope       Scope
	Filter      string
	Attributes  []string
	Subsearches map[string]Subsearch `mapstructure:"joins"`
}

func (s Search) SubsearchAttribute() string {
	keys := maps.Keys(s.Subsearches)
	if 0 == len(keys) {
		return ""
	}
	return keys[0]
}

type Subsearch struct {
	Filter     string
	Scope      Scope
	Attributes []string
}
