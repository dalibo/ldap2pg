package wanted

import (
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slices"
)

type Step struct {
	Description string
	LdapSearch  ldap.Search
	RoleRules   []RoleRule  `mapstructure:"roles"`
	GrantRules  []GrantRule `mapstructure:"grants"`
}

func (s Step) HasLDAPSearch() bool {
	return len(s.LdapSearch.Attributes) > 0
}

func (s Step) HasSubsearch() bool {
	return len(s.LdapSearch.Subsearches) > 0
}

func (s *Step) InferAttributes() {
	attributes := mapset.NewSet[string]()
	subsearchAttributes := make(map[string]mapset.Set[string])

	for field := range s.IterFields() {
		attribute, field, found := strings.Cut(field.FieldName, ".")
		// dn is the primary key of the entry, not a real attribute.
		if "dn" == attribute {
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

	if 0 == attributes.Cardinality() {
		return
	}

	s.LdapSearch.Attributes = attributes.ToSlice()
	slog.Debug("Collected LDAP search attributes.",
		"item", s.Description, "base", s.LdapSearch.Base, "attributes", s.LdapSearch.Attributes)

	if 0 == len(subsearchAttributes) {
		return
	}

	if nil == s.LdapSearch.Subsearches {
		s.LdapSearch.Subsearches = make(map[string]ldap.Subsearch)
	}
	for attribute, subAttributes := range subsearchAttributes {
		subsearch, ok := s.LdapSearch.Subsearches[attribute]
		if !ok {
			subsearch = ldap.Subsearch{
				Filter: "(objectClass=*)",
				Scope:  ldap3.ScopeWholeSubtree,
			}
		}
		subsearch.Attributes = subAttributes.ToSlice()
		slog.Debug("Collected LDAP sub-search attributes.",
			"item", s.Description, "base", s.LdapSearch.Base,
			"fkey", attribute, "attributes", subsearch.Attributes)
		s.LdapSearch.Subsearches[attribute] = subsearch
	}
}

func (s *Step) ReplaceAttributeAsSubentryField() {
	subsearchAttr := s.LdapSearch.SubsearchAttribute()
	for field := range s.IterFields() {
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
func (s Step) IterFields() <-chan *pyfmt.Field {
	ch := make(chan *pyfmt.Field)
	go func() {
		defer close(ch)
		for _, rule := range s.RoleRules {
			for _, f := range rule.Formats() {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
		for _, rule := range s.GrantRules {
			for _, f := range rule.Formats() {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
	}()
	return ch
}

func (s Step) SplitStaticItems() (items []Step) {
	var staticRoles, dynamicRoles []RoleRule
	for _, rule := range s.RoleRules {
		if rule.IsStatic() {
			staticRoles = append(staticRoles, rule)
		} else {
			dynamicRoles = append(dynamicRoles, rule)
		}
	}
	var staticGrants, dynamicGrants []GrantRule
	for _, rule := range s.GrantRules {
		if rule.IsStatic() {
			staticGrants = append(staticGrants, rule)
		} else {
			dynamicGrants = append(dynamicGrants, rule)
		}
	}

	if (0 == len(staticRoles) && 0 == len(staticGrants)) ||
		(0 == len(dynamicRoles) && 0 == len(dynamicGrants)) {
		items = append(items, s)
		return
	}

	items = append(items, Step{
		Description: s.Description,
		LdapSearch:  s.LdapSearch,
		RoleRules:   dynamicRoles,
		GrantRules:  dynamicGrants,
	})

	items = append(items, Step{
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
func (s Step) search(ldapc ldap.Client) <-chan SearchResult {
	ch := make(chan SearchResult)
	go func() {
		defer close(ch)
		if !s.HasLDAPSearch() {
			// Use a dumb empty result.
			ch <- SearchResult{}
			return
		}

		search := s.LdapSearch
		res, err := ldapc.Search(search.Base, search.Scope, search.Filter, search.Attributes)
		if err != nil {
			ch <- SearchResult{err: err}
			return
		}
		subsearchAttr := s.LdapSearch.SubsearchAttribute()
		for _, entry := range res.Entries {
			slog.Debug("Got LDAP entry.", "dn", entry.DN)
			result := ldap.Result{
				Entry:              entry,
				SubsearchAttribute: subsearchAttr,
			}
			if "" == subsearchAttr {
				ch <- SearchResult{result: result}
				continue
			}
			bases := entry.GetEqualFoldAttributeValues(subsearchAttr)
			for _, base := range bases {
				s := s.LdapSearch.Subsearches[subsearchAttr]
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

func (s Step) generateRoles(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
	go func() {
		defer close(ch)
		for _, rule := range s.RoleRules {
			for role := range rule.Generate(results) {
				ch <- role
			}
		}
	}()
	return ch
}

func (s Step) generateGrants(results *ldap.Result, privs privileges.RefMap) <-chan privileges.Grant {
	ch := make(chan privileges.Grant)
	go func() {
		defer close(ch)
		for _, rule := range s.GrantRules {
			for grant := range rule.Generate(results, privs) {
				grant.Normalize()
				ch <- grant
			}
		}
	}()
	return ch
}
