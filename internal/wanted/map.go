package wanted

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/privileges"
	"github.com/dalibo/ldap2pg/internal/role"
)

// Rules holds a set of rules to generate wanted state.
type Rules []Step

func (m Rules) HasLDAPSearches() bool {
	for _, item := range m {
		if item.HasLDAPSearch() {
			return true
		}
	}
	return false
}

func (m Rules) SplitStaticRules() (newMap Rules) {
	newMap = make(Rules, 0)
	for _, item := range m {
		newMap = append(newMap, item.SplitStaticItems()...)
	}
	return
}

func (m Rules) DropGrants() (out Rules) {
	out = make(Rules, 0)
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

func (m Rules) Run(blacklist lists.Blacklist) (roles role.Map, grants map[string][]privileges.Grant, err error) {
	var errList []error
	var ldapc ldap.Client
	if m.HasLDAPSearches() {
		err := ldap.Initialize()
		if err != nil {
			return nil, nil, err
		}

		ldapc, err = ldap.Connect()
		if err != nil {
			return nil, nil, err
		}
		defer ldapc.Conn.Close()
	}

	roles = make(map[string]role.Role)
	grants = make(map[string][]privileges.Grant)
	for i, item := range m {
		if item.Description != "" {
			slog.Info(item.Description)
		} else {
			slog.Debug(fmt.Sprintf("Processing sync map item %d.", i))
		}

		for res := range item.search(ldapc) {
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
						"parents", role.Parents, "comment", role.Comment)
				} else {
					slog.Debug("Wants role.",
						"name", role.Name, "options", role.Options,
						"parents", role.Parents, "comment", role.Comment)
				}
				roles[role.Name] = role
			}

			for grant := range item.generateGrants(&res.result) {
				pattern := blacklist.MatchString(grant.Grantee)
				if pattern != "" {
					slog.Debug(
						"Ignoring grant to blacklisted role.",
						"to", grant.Grantee, "pattern", pattern)
					continue
				}
				_, exists := roles[grant.Grantee]
				if !exists {
					slog.Error("Generated grant on unwanted role.", "grant", grant, "role", grant.Grantee)
					errList = append(errList, fmt.Errorf("grant on unknown role"))
					continue
				}
				acl := grant.ACLName()
				slog.Debug("Wants grant.", "grant", grant, "acl", acl)
				grants[acl] = append(grants[acl], grant)
			}
		}
	}

	if 0 < len(errList) {
		err = errors.Join(errList...)
	}
	return
}
