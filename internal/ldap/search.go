package ldap

import (
	"maps"
	"slices"
)

type Search struct {
	Base        string
	Scope       Scope
	Filter      string
	Attributes  []string
	Subsearches map[string]Subsearch `mapstructure:"joins"`
}

func (s Search) SubsearchAttribute() string {
	keys := slices.Collect(maps.Keys(s.Subsearches))
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
