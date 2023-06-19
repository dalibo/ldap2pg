package wanted

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/role"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type Item struct {
	Description string
	LdapSearch  ldap.Search
	RoleRules   []RoleRule  `mapstructure:"roles"`
	GrantRules  []GrantRule `mapstructure:"grants"`
}

func (i Item) HasLDAPSearch() bool {
	return 0 < len(i.LdapSearch.Attributes)
}

func (i Item) HasSubsearch() bool {
	return 0 < len(i.LdapSearch.Subsearches)
}

var knownRDN = []string{"cn", "l", "st", "o", "ou", "c", "street", "dc", "uid"}

func (i *Item) InferAttributes() {
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
	slog.Debug("Collected LDAP search attributes.",
		"item", i.Description, "base", i.LdapSearch.Base, "attributes", i.LdapSearch.Attributes)

	if 0 == len(subsearchAttributes) {
		return
	}

	if nil == i.LdapSearch.Subsearches {
		i.LdapSearch.Subsearches = make(map[string]ldap.Subsearch)
	}
	for attribute, subAttributes := range subsearchAttributes {
		subsearch, ok := i.LdapSearch.Subsearches[attribute]
		if !ok {
			subsearch = ldap.Subsearch{
				Filter: "(objectClass=*)",
				Scope:  ldap3.ScopeWholeSubtree,
			}
		}
		subsearch.Attributes = subAttributes.ToSlice()
		slog.Debug("Collected LDAP sub-search attributes.",
			"item", i.Description, "base", i.LdapSearch.Base,
			"fkey", attribute, "attributes", subsearch.Attributes)
		i.LdapSearch.Subsearches[attribute] = subsearch
	}
}

func (i *Item) ReplaceAttributeAsSubentryField() {
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
func (i Item) IterFields() <-chan *pyfmt.Field {
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
		for _, rule := range i.GrantRules {
			allFormats := []pyfmt.Format{
				rule.Privilege, rule.Database, rule.Schema, rule.Object, rule.To,
			}

			for _, f := range allFormats {
				for _, field := range f.Fields {
					ch <- field
				}
			}
		}
	}()
	return ch
}

func (i Item) SplitStaticItems() (items []Item) {
	var staticRoles, dynamicRoles []RoleRule
	for _, rule := range i.RoleRules {
		if rule.IsStatic() {
			staticRoles = append(staticRoles, rule)
		} else {
			dynamicRoles = append(dynamicRoles, rule)
		}
	}
	var staticGrants, dynamicGrants []GrantRule
	for _, rule := range i.GrantRules {
		if rule.IsStatic() {
			staticGrants = append(staticGrants, rule)
		} else {
			dynamicGrants = append(dynamicGrants, rule)
		}
	}

	if (0 == len(staticRoles) && 0 == len(staticGrants)) ||
		(0 == len(dynamicRoles) && 0 == len(dynamicGrants)) {
		items = append(items, i)
		return
	}

	items = append(items, Item{
		Description: i.Description,
		LdapSearch:  i.LdapSearch,
		RoleRules:   dynamicRoles,
		GrantRules:  dynamicGrants,
	})

	items = append(items, Item{
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
func (i Item) search(ldapc ldap.Client, watch *perf.StopWatch) <-chan SearchResult {
	ch := make(chan SearchResult)
	go func() {
		defer close(ch)
		if !i.HasLDAPSearch() {
			// Use a dumb empty result.
			ch <- SearchResult{}
			return
		}

		s := i.LdapSearch
		res, err := ldapc.Search(watch, s.Base, s.Scope, s.Filter, s.Attributes)
		if err != nil {
			ch <- SearchResult{err: err}
			return
		}
		subsearchAttr := i.LdapSearch.SubsearchAttribute()
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
			bases := entry.GetAttributeValues(subsearchAttr)
			for _, base := range bases {
				s := i.LdapSearch.Subsearches[subsearchAttr]
				res, err = ldapc.Search(watch, base, s.Scope, s.Filter, s.Attributes)
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

func (i Item) generateRoles(results *ldap.Result) <-chan role.Role {
	ch := make(chan role.Role)
	go func() {
		defer close(ch)
		for _, rule := range i.RoleRules {
			for role := range rule.Generate(results) {
				ch <- role
			}
		}
	}()
	return ch
}

func (i Item) generateGrants(results *ldap.Result, privileges privilege.RefMap) <-chan privilege.Grant {
	ch := make(chan privilege.Grant)
	go func() {
		defer close(ch)
		for _, rule := range i.GrantRules {
			for grant := range rule.Generate(results, privileges) {
				ch <- grant
			}
		}
	}()
	return ch
}
