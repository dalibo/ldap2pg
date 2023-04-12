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

func (rule *RoleRule) LoadYaml(yaml map[string]interface{}) {
	rule.Names = yaml["names"].([]string)
	rule.Comments = yaml["comments"].([]string)
}
