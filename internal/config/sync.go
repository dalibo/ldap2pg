package config

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/pyfmt"
	mapset "github.com/deckarep/golang-set/v2"
)

type SyncItem struct {
	Description string
	LdapSearch  LdapSearch
	RoleRules   []RoleRule `mapstructure:"roles"`
}

func (i SyncItem) ListAttributes() []string {
	return nil
}

func (i SyncItem) HasLDAPSearch() bool {
	return 0 < len(i.LdapSearch.Attributes)
}

func (i *SyncItem) InferAttributes() {
	attributes := mapset.NewSet[string]()
	for _, rule := range i.RoleRules {
		listOfLists := []interface{}{
			[]pyfmt.Format{rule.Comment},
			rule.Names,
			rule.Parents,
		}
		for _, item := range listOfLists {
			list := item.([]pyfmt.Format)
			for _, f := range list {
				for _, field := range f.Fields {
					attribute, _, _ := strings.Cut(field.FieldName, ".")
					if "dn" == attribute {
						continue
					}
					attributes.Add(attribute)
				}
			}
		}
	}
	i.LdapSearch.Attributes = attributes.ToSlice()
}
