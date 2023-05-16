// Logic to describe wanted state from YAML and LDAP
package states

import (
	"errors"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/dalibo/ldap2pg/internal/search"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleMap
}

func ComputeWanted(watch *perf.StopWatch, syncMap search.SyncMap, blacklist lists.Blacklist) (wanted Wanted, err error) {
	var errList []error
	var ldapc ldap.Client
	if syncMap.HasLDAPSearches() {
		ldapOptions, err := ldap.Initialize()
		if err != nil {
			return wanted, err
		}

		ldapc, err = ldap.Connect(ldapOptions)
		if err != nil {
			return wanted, err
		}
		defer ldapc.Conn.Close()
	}

	wanted.Roles = make(map[string]roles.Role)
	for _, item := range syncMap {
		if item.Description != "" {
			slog.Info(item.Description)
		} else {
			slog.Debug("Next sync map item.")
		}

		for data := range SearchDirectory(ldapc, watch, item) {
			err, failed := data.(error)
			if failed {
				slog.Error("Search error. Keep going.", "err", err)
				errList = append(errList, err)
				continue
			}

			for role := range GenerateAllRoles(item.RoleRules, data.(*ldap.Results)) {
				if "" == role.Name {
					continue
				}
				pattern := blacklist.MatchString(role.Name)
				if pattern != "" {
					slog.Debug(
						"Ignoring blacklisted wanted role.",
						"role", role.Name, "pattern", pattern)
					continue
				}
				_, exists := wanted.Roles[role.Name]
				if exists {
					slog.Warn("Duplicated wanted role.", "role", role.Name)
				}
				slog.Debug("Wants role.",
					"name", role.Name, "options", role.Options, "parents", role.Parents, "comment", role.Comment)
				wanted.Roles[role.Name] = role
			}
		}
	}
	if 0 < len(errList) {
		err = errors.Join(errList...)
	}
	return
}

// Search directory, returning each entry or error. Sub-searches are done
// concurrently and returned for each sub-key.
func SearchDirectory(ldapc ldap.Client, watch *perf.StopWatch, item search.SyncItem) <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
		if !item.HasLDAPSearch() {
			// Use a dumb empty result.
			ch <- &ldap.Results{}
			return
		}

		s := item.LdapSearch
		res, err := ldapc.Search(watch, s.Base, s.Scope, s.Filter, s.Attributes)
		if err != nil {
			ch <- err
			return
		}
		subsearchAttr := item.LdapSearch.SubsearchAttribute()
		for _, entry := range res.Entries {
			slog.Debug("Got LDAP entry.", "dn", entry.DN)
			results := ldap.Results{
				Entry:              entry,
				SubsearchAttribute: subsearchAttr,
			}
			if "" == subsearchAttr {
				ch <- &results
				continue
			}
			bases := entry.GetAttributeValues(subsearchAttr)
			for _, base := range bases {
				s := item.LdapSearch.Subsearches[subsearchAttr]
				res, err = ldapc.Search(watch, base, s.Scope, s.Filter, s.Attributes)
				if err != nil {
					ch <- err
					continue
				}
				// Copy results in scope.
				results := results
				// Overwrite previous sub-entries and resend results.
				results.SubsearchEntries = res.Entries
				ch <- &results
			}
		}
	}()
	return ch
}

func GenerateAllRoles(rules []search.RoleRule, results *ldap.Results) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		for _, rule := range rules {
			for role := range GenerateRoles(rule, results) {
				ch <- role
			}
		}
	}()
	return ch
}

func GenerateRoles(rule search.RoleRule, results *ldap.Results) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		var parents []string
		for _, f := range rule.Parents {
			if nil == results.Entry || 0 == len(f.Fields) {
				// Static case.
				parents = append(parents, f.String())
			} else {
				// Dynamic case.
				for values := range results.GenerateValues(f) {
					parents = append(parents, f.Format(values))
				}
			}
		}

		if nil == results.Entry {
			// Case static role.
			role := roles.Role{}
			role.Name = rule.Name.String()
			role.Comment = rule.Comment.String()
			role.Options = rule.Options
			role.Parents = mapset.NewSet[string](parents...)
			ch <- role
		} else {
			// Case dynamic roles.
			for values := range results.GenerateValues(rule.Name, rule.Comment) {
				role := roles.Role{}
				role.Name = rule.Name.Format(values)
				role.Comment = rule.Comment.Format(values)
				role.Options = rule.Options
				role.Parents = mapset.NewSet[string](parents...)
				ch <- role
			}
		}
	}()
	return ch
}
