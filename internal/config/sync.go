package config

type SyncItem struct {
	Description string
	LdapSearch  LdapSearch
	RoleRules   []RoleRule `mapstructure:"roles"`
}

func (i SyncItem) ListAttributes() []string {
	return nil
}
