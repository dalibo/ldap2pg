package config

import (
	mapset "github.com/deckarep/golang-set/v2"
)

type RoleRule struct {
	Names    []string
	Options  RoleOptions
	Comments []string
	Parents  mapset.Set[string]
}

func (rule *RoleRule) LoadYaml(yaml map[string]interface{}) {
	rule.Names = yaml["names"].([]string)
	rule.Comments = yaml["comments"].([]string)
	rule.Options.LoadYaml(yaml["options"].(map[string]interface{}))
	rule.Parents = mapset.NewSet[string](yaml["parents"].([]string)...)
}
