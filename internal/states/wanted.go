// Logic to describe wanted state from YAML and LDAP
package states

import (
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
	var ldapConn *ldap3.Conn
	if config.HasLDAPSearches() {
		ldapOptions, err := ldap.Initialize()
		if err != nil {
			return wanted, err
		}

		ldapConn, err = ldap.Connect(ldapOptions)
		if err != nil {
			return wanted, err
		}
		defer ldapConn.Close()
	}

	wanted.Roles = make(map[string]roles.Role)
	for _, item := range config.SyncItems {
		var entries []*ldap3.Entry
		if item.Description != "" {
			slog.Info(item.Description)
		}

		if item.HasLDAPSearch() {
			search := ldap3.SearchRequest{
				BaseDN:     item.LdapSearch.Base,
				Scope:      ldap3.ScopeWholeSubtree,
				Filter:     item.LdapSearch.Filter,
				Attributes: item.LdapSearch.Attributes,
			}
			slog.Debug("Searching LDAP directory.",
				"base", search.BaseDN, "filter", search.Filter, "attributes", search.Attributes)

			var res *ldap3.SearchResult
			duration := timer.TimeIt(func() {
				res, err = ldapConn.Search(&search)
			})
			if err != nil {
				return wanted, err
			}
			entries = res.Entries
			slog.Debug("LDAP search done.", "duration", duration, "entries", len(res.Entries))
		} else {
			entries = [](*ldap3.Entry){nil}
		}

		for _, rule := range item.RoleRules {
			for _, entry := range entries {
				for role := range GenerateRoles(rule, entry) {
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
						"name", role.Name, "options", role.Options, "parents", role.Parents)
					wanted.Roles[role.Name] = role
				}
			}
		}
	}
	return
}

func GenerateRoles(rule config.RoleRule, entry *ldap3.Entry) <-chan roles.Role {
	ch := make(chan roles.Role)
	go func() {
		defer close(ch)
		var parents []string
		for _, f := range rule.Parents {
			if nil == entry || 0 == len(f.Fields) {
				// Static case.
				parents = append(parents, f.String())
			} else {
				// Dynamic case.
				for values := range ldap.GenerateValues(entry, f) {
					parents = append(parents, f.Format(values))
				}
			}
		}

		if nil == entry {
			// Case static role.
			role := roles.Role{}
			role.Name = rule.Name.String()
			role.Comment = rule.Comment.String()
			role.Options = rule.Options
			role.Parents = mapset.NewSet[string](parents...)
			ch <- role
		} else {
			// Case dynamic roles.
			for values := range ldap.GenerateValues(entry, rule.Name, rule.Comment) {
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
