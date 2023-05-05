package config

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type SyncItem struct {
	Description string
	LdapSearch  LdapSearch
	RoleRules   []RoleRule `mapstructure:"roles"`
}

func (i SyncItem) HasLDAPSearch() bool {
	return 0 < len(i.LdapSearch.Attributes)
}

func (i SyncItem) HasSubsearch() bool {
	return 0 < len(i.LdapSearch.Subsearches)
}

func (s LdapSearch) SubsearchAttribute() string {
	keys := maps.Keys(s.Subsearches)
	if 0 == len(keys) {
		return ""
	}
	return keys[0]
}

var knownRDN = []string{"cn", "l", "st", "o", "ou", "c", "street", "dc", "uid"}

func (i *SyncItem) InferAttributes() {
	attributes := mapset.NewSet[string]()
	subsearchAttributes := make(map[string]mapset.Set[string])

	for field := range i.IterFields() {
		attribute, field, found := strings.Cut(field.FieldName, ".")
		// dn is the primary key of the entry, not a real attribute.
		if "dn" == attribute {
			continue
		}
		attributes.Add(attribute)

		// Case {member} or {member.cn}
		if !found || slices.Contains[string](knownRDN, field) {
			continue
		}

		// Case {member.SAMAccountName}
		subAttributes, ok := subsearchAttributes[attribute]
		if !ok {
			subAttributes = mapset.NewSet[string]()
		}
		subAttributes.Add(field)
		subsearchAttributes[attribute] = subAttributes
	}

	if 0 == attributes.Cardinality() {
		return
	}

	i.LdapSearch.Attributes = attributes.ToSlice()
	if "" == i.LdapSearch.Filter {
		i.LdapSearch.Filter = "(objectClass=*)"
	}
	i.LdapSearch.Filter = ldap.CleanFilter(i.LdapSearch.Filter)
	slog.Debug("Collected LDAP search attributes.",
		"item", i.Description, "base", i.LdapSearch.Base, "attributes", i.LdapSearch.Attributes)

	if 0 == len(subsearchAttributes) {
		return
	}

	if nil == i.LdapSearch.Subsearches {
		i.LdapSearch.Subsearches = make(map[string]Subsearch)
	}
	for attribute, subAttributes := range subsearchAttributes {
		subsearch, ok := i.LdapSearch.Subsearches[attribute]
		if !ok {
			subsearch = Subsearch{
				Scope: ldap3.ScopeWholeSubtree,
			}
		}
		subsearch.Attributes = subAttributes.ToSlice()
		if "" == subsearch.Filter {
			subsearch.Filter = "(objectClass=*)"
		}
		subsearch.Filter = ldap.CleanFilter(subsearch.Filter)
		slog.Debug("Collected LDAP sub-search attributes.",
			"item", i.Description, "base", i.LdapSearch.Base,
			"fkey", attribute, "attributes", subsearch.Attributes)
		i.LdapSearch.Subsearches[attribute] = subsearch
	}
}

func (i *SyncItem) ReplaceAttributeAsSubentryField() {
	subsearchAttr := i.LdapSearch.SubsearchAttribute()
	for field := range i.IterFields() {
		attribute, _, found := strings.Cut(field.FieldName, ".")
		if attribute != subsearchAttr {
			continue
		}
		// When sub-searching, never use sub attribute directly but
		// always use sub-entry attributes. This avoid double product
		// when computing combinations.
		if !found {
			// Case {member} -> {member.dn}
			field.FieldName = attribute + ".dn"
			continue
		}
	}
}

// Yields all {attr} from all formats in item.
func (i SyncItem) IterFields() <-chan *pyfmt.Field {
	ch := make(chan *pyfmt.Field)
	go func() {
		defer close(ch)
		for _, rule := range i.RoleRules {
			allFormats := []pyfmt.Format{
				rule.Name, rule.Comment,
			}
			allFormats = append(allFormats, rule.Parents...)

			for _, f := range allFormats {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
	}()
	return ch
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
