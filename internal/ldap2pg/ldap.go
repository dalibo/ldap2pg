package ldap2pg

import (
	"time"

	ldap "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

func LdapConnect(config Config) (err error) {
	slog.Info("Connecting to LDAP directory.",
		"uri", config.Ldap.URI,
		"binddn", config.Ldap.BindDn)

	slog.Debug("LDAP dial.",
		"uri", config.Ldap.URI)

	var ldapconn *ldap.Conn
	for try := 0; try < 15; try++ {
		ldapconn, err = ldap.DialURL(config.Ldap.URI)
		if err != nil {
			slog.Debug("Retrying LDAP connection in 1s.",
				"error", err)

			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return
	}

	defer ldapconn.Close()
	slog.Debug("LDAP simple bind.",
		"binddn", config.Ldap.BindDn)

	err = ldapconn.Bind(config.Ldap.BindDn, config.Ldap.Password)
	if err != nil {
		return
	}

	slog.Debug("Running LDAP whoami.")
	wai, err := ldapconn.WhoAmI(nil)
	if err != nil {
		return
	}
	slog.Debug("LDAP whoami done.",
		"authzid", wai.AuthzID)

	return
}
