// Logic to describe wanted state from YAML and LDAP
package states

import (
	"errors"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/dalibo/ldap2pg/internal/utils"
	mapset "github.com/deckarep/golang-set/v2"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

type Wanted struct {
	Roles roles.RoleMap
}

func ComputeWanted(timer *utils.Timer, config config.Config, blacklist utils.Blacklist) (wanted Wanted, err error) {
	var errList []error
	var ldapc ldap.Client
	if config.HasLDAPSearches() {
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
	for _, item := range config.SyncItems {
		if item.Description != "" {
			slog.Info(item.Description)
		}

		for data := range SearchDirectory(ldapc, timer, item) {
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
func SearchDirectory(ldapc ldap.Client, timer *utils.Timer, item config.SyncItem) <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
		if !item.HasLDAPSearch() {
			// Use a dumb empty result.
			ch <- &ldap.Results{}
			return
		}

		search := ldap3.SearchRequest{
			BaseDN:     item.LdapSearch.Base,
			Scope:      ldap3.ScopeWholeSubtree,
			Filter:     item.LdapSearch.Filter,
			Attributes: item.LdapSearch.Attributes,
		}
		args := []string{"-b", search.BaseDN, "-s", ldap.ScopeArg(search.Scope), search.Filter}
		args = append(args, search.Attributes...)
		slog.Debug("Searching LDAP directory.", "cmd", ldapc.Command("ldapsearch", args...))

		var res *ldap3.SearchResult
		var err error
		duration := timer.TimeIt(func() {
			res, err = ldapc.Conn.Search(&search)
		})
		if err != nil {
			slog.Debug("LDAP search failed.", "duration", duration, "err", err)
			ch <- err
			return
		}
		slog.Debug("LDAP search done.", "duration", duration, "entries", len(res.Entries))

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
				search := ldap3.SearchRequest{
					BaseDN:     base,
					Scope:      ldap3.ScopeBaseObject,
					Filter:     item.LdapSearch.Subsearches[subsearchAttr].Filter,
					Attributes: item.LdapSearch.Subsearches[subsearchAttr].Attributes,
				}
				args := []string{"-b", search.BaseDN, "-s", ldap.ScopeArg(search.Scope), search.Filter}
				args = append(args, search.Attributes...)
				slog.Debug("Recursive LDAP search.", "cmd", ldapc.Command("ldapsearch", args...))
				duration := timer.TimeIt(func() {
					res, err = ldapc.Conn.Search(&search)
				})
				if err != nil {
					slog.Debug("LDAP search failed.", "duration", duration, "err", err)
					ch <- err
					continue
				}
				slog.Debug("LDAP search done.", "duration", duration, "entries", len(res.Entries))
				// Overwrite previous sub-entries and resend results.
				results.SubsearchEntries = res.Entries
				ch <- &results
			}
		}
	}()
	return ch
}

func GenerateAllRoles(rules []config.RoleRule, results *ldap.Results) <-chan roles.Role {
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

func GenerateRoles(rule config.RoleRule, results *ldap.Results) <-chan roles.Role {
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
