// Logic to describe wanted state from YAML and LDAP
package sync

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

type Wanted struct {
	Roles  role.Map
	Grants []privilege.Grant
}

func (syncMap Map) Wanted(watch *perf.StopWatch, blacklist lists.Blacklist, privileges privilege.RefMap) (wanted Wanted, err error) {
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

	wanted.Roles = make(map[string]role.Role)
	for i, item := range syncMap {
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
				_, exists := wanted.Roles[role.Name]
				if exists {
					slog.Warn("Duplicated wanted role.", "role", role.Name)
				}
				slog.Debug("Wants role.",
					"name", role.Name, "options", role.Options, "parents", role.Parents, "comment", role.Comment)
				wanted.Roles[role.Name] = role
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
				wanted.Grants = append(wanted.Grants, grant)
			}
		}
	}
	if 0 < len(errList) {
		err = errors.Join(errList...)
	}
	return
}
