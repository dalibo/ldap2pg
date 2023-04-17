package config

type SyncItem struct {
	Description string
	LdapSearch  interface{}
	RoleRules   []RoleRule
}

func (item *SyncItem) LoadYaml(yaml map[string]interface{}) (err error) {
	desc, ok := yaml["description"]
	if ok {
		item.Description = desc.(string)
	}
	rules, ok := yaml["roles"]
	if ok {
		ruleList := rules.([]interface{})
		for _, yamlRule := range ruleList {
			rule := RoleRule{}
			// Default Inherit like Postgres.
			rule.Options.Inherit = true
			// Default ConnLimit like Postgres.
			rule.Options.ConnLimit = -1
			yamlRuleMap := yamlRule.(map[string]interface{})
			rule.LoadYaml(yamlRuleMap)
			item.RoleRules = append(item.RoleRules, rule)
		}
	}
	iLdap, exists := yaml["ldapsearch"]
	if exists {
		item.LdapSearch = iLdap
	}
	return
}
