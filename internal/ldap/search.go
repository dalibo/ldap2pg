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
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

type Subsearch struct {
	Filter     string
	Scope      Scope
	Attributes []string
}
