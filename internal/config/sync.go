package config

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
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
		allFormats := []pyfmt.Format{
			rule.Name, rule.Comment,
		}
		allFormats = append(allFormats, rule.Parents...)
		for _, f := range allFormats {
			for _, field := range f.Fields {
				attribute, _, _ := strings.Cut(field.FieldName, ".")
				if "dn" == attribute {
					continue
				}
				attributes.Add(attribute)
			}
		}
	}
	if 0 == attributes.Cardinality() {
		return
	}
	i.LdapSearch.Attributes = attributes.ToSlice()
	if "" == i.LdapSearch.Filter {
		i.LdapSearch.Filter = "(objectClass=*)"
	}
	i.LdapSearch.Filter = ldap.CleanFilter(i.LdapSearch.Filter)
}

func (i SyncItem) SplitStaticItems() (items []SyncItem) {
	var staticRules, dynamicRules []RoleRule
	for _, rule := range i.RoleRules {
		if rule.IsStatic() {
			staticRules = append(staticRules, rule)
		} else {
			dynamicRules = append(dynamicRules, rule)
		}
	}

	if len(staticRules) == 0 || len(dynamicRules) == 0 {
		items = append(items, i)
		return
	}

	items = append(items, SyncItem{
		Description: i.Description,
		LdapSearch:  i.LdapSearch,
		RoleRules:   dynamicRules,
	})

	items = append(items, SyncItem{
		// Avoid duplicating log message, use a silent item.
		Description: "",
		RoleRules:   staticRules,
	})

	return
}
