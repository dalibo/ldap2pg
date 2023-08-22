package wanted

import (
	"errors"
	"fmt"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/privilege"
	"github.com/dalibo/ldap2pg/internal/role"
	"golang.org/x/exp/slog"
)

// Map holds a set of rules to generate wanted state.
type Map []Item

func (m Map) HasLDAPSearches() bool {
	for _, item := range m {
		if item.HasLDAPSearch() {
			return true
		}
	}
	return false
}

func (m Map) SplitStaticRules() (newMap Map) {
	newMap = make(Map, 0)
	for _, item := range m {
		newMap = append(newMap, item.SplitStaticItems()...)
	}
	return
}

func (m Map) DropGrants() (out Map) {
	out = make(Map, 0)
	for _, item := range m {
		item.GrantRules = nil
		if 0 < len(item.RoleRules) {
			out = append(out, item)
		} else {
			slog.Debug("Dropping sync map item with grants.", "item", item)
		}
	}
	return
}

func (m Map) Run(watch *perf.StopWatch, blacklist lists.Blacklist, privileges privilege.RefMap) (roles role.Map, grants []privilege.Grant, err error) {
	var errList []error
	var ldapc ldap.Client
	if m.HasLDAPSearches() {
		ldapOptions, err := ldap.Initialize()
		if err != nil {
			return nil, nil, err
		}

		ldapc, err = ldap.Connect(ldapOptions)
		if err != nil {
			return nil, nil, err
		}
		defer ldapc.Conn.Close()
	}

	roles = make(map[string]role.Role)
	for i, item := range m {
		if item.Description != "" {
			slog.Info(item.Description)
		} else {
			slog.Debug(fmt.Sprintf("Processing sync map item %d.", i))
		}

		for res := range item.search(ldapc, watch) {
			if res.err != nil {
				slog.Error("Search error. Keep going.", "err", res.err)
				errList = append(errList, res.err)
				continue
			}

			for role := range item.generateRoles(&res.result) {
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
				current, exists := roles[role.Name]
				if exists {
					current.Merge(role)
					role = current
					slog.Debug("Updated wanted role.",
						"name", role.Name, "options", role.Options,
						"parents", role.Parents.ToSlice(), "comment", role.Comment)
				} else {
					slog.Debug("Wants role.",
						"name", role.Name, "options", role.Options,
						"parents", role.Parents.ToSlice(), "comment", role.Comment)
				}
				roles[role.Name] = role
			}

			for grant := range item.generateGrants(&res.result, privileges) {
				pattern := blacklist.MatchString(grant.Grantee)
				if pattern != "" {
					slog.Debug(
						"Ignoring grant to blacklisted role.",
						"to", grant.Grantee, "pattern", pattern)
					continue
				}
				slog.Debug("Wants grant.", "grant", grant)
				grants = append(grants, grant)
			}
		}
	}
	if 0 < len(errList) {
		err = errors.Join(errList...)
	}
	return
}
