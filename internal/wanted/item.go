package wanted

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/dalibo/ldap2pg/v6/internal/ldap"
	"github.com/dalibo/ldap2pg/v6/internal/privileges"
	"github.com/dalibo/ldap2pg/v6/internal/pyfmt"
	"github.com/dalibo/ldap2pg/v6/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
)

type RulesItem struct {
	pos         int
	Description string
	LdapSearch  ldap.Search
	RoleRules   []RoleRule             `mapstructure:"roles"`
	GrantRules  []privileges.GrantRule `mapstructure:"grants"`
}

func (item RulesItem) String() string {
	if item.Description == "" {
		return fmt.Sprintf("%d", item.pos)
	}
	return item.Description
}

func (item RulesItem) HasLDAPSearch() bool {
	return len(item.LdapSearch.Attributes) > 0
}

func (item RulesItem) HasSubsearch() bool {
	return len(item.LdapSearch.Subsearches) > 0
}

func (item *RulesItem) InferAttributes() {
	attributes := mapset.NewSet[string]()
	subsearchAttributes := make(map[string]mapset.Set[string])

	for field := range item.IterFields() {
		attribute, field, found := strings.Cut(field.FieldName, ".")
		// dn is the primary key of the entry, not a real attribute.
		if attribute == "dn" {
			continue
		}
		attributes.Add(attribute)

		// Case {member} or {member.cn}
		if !found || slices.Contains(ldap.KnownRDNs, field) {
			continue
		}

		// Case {member.sAMAccountName}
		subAttributes, ok := subsearchAttributes[attribute]
		if !ok {
			subAttributes = mapset.NewSet[string]()
		}
		subAttributes.Add(field)
		subsearchAttributes[attribute] = subAttributes
	}

	if attributes.Cardinality() == 0 {
		return
	}

	item.LdapSearch.Attributes = attributes.ToSlice()
	slog.Debug("Collected LDAP search attributes.",
		"item", item, "base", item.LdapSearch.Base, "attributes", item.LdapSearch.Attributes)

	if len(subsearchAttributes) == 0 {
		return
	}

	if item.LdapSearch.Subsearches == nil {
		item.LdapSearch.Subsearches = make(map[string]ldap.Subsearch)
	}
	for attribute, subAttributes := range subsearchAttributes {
		subsearch, ok := item.LdapSearch.Subsearches[attribute]
		if !ok {
			subsearch = ldap.Subsearch{
				Filter: "(objectClass=*)",
				Scope:  ldap3.ScopeWholeSubtree,
			}
		}
		subsearch.Attributes = subAttributes.ToSlice()
		slog.Debug("Collected LDAP sub-search attributes.",
			"item", item, "base", item.LdapSearch.Base,
			"fkey", attribute, "attributes", subsearch.Attributes)
		item.LdapSearch.Subsearches[attribute] = subsearch
	}
}

func (item *RulesItem) ReplaceAttributeAsSubentryField() {
	subsearchAttr := item.LdapSearch.SubsearchAttribute()
	for field := range item.IterFields() {
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
func (item RulesItem) IterFields() <-chan *pyfmt.Field {
	ch := make(chan *pyfmt.Field)
	go func() {
		defer close(ch)
		for _, rule := range item.RoleRules {
			for _, f := range rule.Formats() {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
		for _, rule := range item.GrantRules {
			for _, f := range rule.Formats() {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
	}()
	return ch
}

func (item RulesItem) SplitStaticItems() (items []RulesItem) {
	var staticRoles, dynamicRoles []RoleRule
	for _, rule := range item.RoleRules {
		if rule.IsStatic() {
			staticRoles = append(staticRoles, rule)
		} else {
			dynamicRoles = append(dynamicRoles, rule)
		}
	}
	var staticGrants, dynamicGrants []privileges.GrantRule
	for _, rule := range item.GrantRules {
		if rule.IsStatic() {
			staticGrants = append(staticGrants, rule)
		} else {
			dynamicGrants = append(dynamicGrants, rule)
		}
	}

	if (len(staticRoles) == 0 && len(staticGrants) == 0) ||
		(len(dynamicRoles) == 0 && len(dynamicGrants) == 0) {
		items = append(items, item)
		return
	}

	items = append(items, RulesItem{
		Description: item.Description,
		LdapSearch:  item.LdapSearch,
		RoleRules:   dynamicRoles,
		GrantRules:  dynamicGrants,
	})

	items = append(items, RulesItem{
		// Avoid duplicating log message, use a silent item.
		Description: "",
		RoleRules:   staticRoles,
		GrantRules:  staticGrants,
	})

	return
}

type SearchResult struct {
	result ldap.Result
	err    error
}

// search directory, returning each entry or error. Sub-searches are done
// concurrently and returned for each sub-key.
func (item RulesItem) search(ldapc ldap.Client) <-chan SearchResult {
	ch := make(chan SearchResult)
	go func() {
		defer close(ch)
		if !item.HasLDAPSearch() {
			// Use a dumb empty result.
			ch <- SearchResult{}
			return
		}

		search := item.LdapSearch
		res, err := ldapc.Search(search.Base, search.Scope, search.Filter, search.Attributes)
		if err != nil {
			ch <- SearchResult{err: err}
			return
		}
		subsearchAttr := item.LdapSearch.SubsearchAttribute()
		for _, entry := range res.Entries {
			slog.Debug("Got LDAP entry.", "dn", entry.DN)
			result := ldap.Result{
				Entry:              entry,
				SubsearchAttribute: subsearchAttr,
			}
			if subsearchAttr == "" {
				ch <- SearchResult{result: result}
				continue
			}
			bases := entry.GetEqualFoldAttributeValues(subsearchAttr)
			for _, base := range bases {
				s := item.LdapSearch.Subsearches[subsearchAttr]
				res, err = ldapc.Search(base, s.Scope, s.Filter, s.Attributes)
				if err != nil {
					ch <- SearchResult{err: err}
					continue
				}
				// Copy results in scope.
				result := result
				// Overwrite previous sub-entries and resend results.
				result.SubsearchEntries = res.Entries
				ch <- SearchResult{result: result}
			}
		}
	}()
	return ch
}

func (item RulesItem) generateRoles(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
	go func() {
		defer close(ch)
		for _, rule := range item.RoleRules {
			for role := range rule.Generate(results) {
				ch <- role
			}
		}
	}()
	return ch
}

func (item RulesItem) generateGrants(results *ldap.Result) <-chan privileges.Grant {
	ch := make(chan privileges.Grant)
	go func() {
		defer close(ch)
		for _, rule := range item.GrantRules {
			for grant := range rule.Generate(results) {
				ch <- grant
			}
		}
	}()
	return ch
}
