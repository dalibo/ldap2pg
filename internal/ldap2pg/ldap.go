package ldap2pg

import (
	"time"

	ldap "github.com/go-ldap/ldap/v3"
)

func LdapConnect(config Config) (err error) {
	Logger.Infow("Connecting to LDAP directory.", "uri", config.Ldap.URI, "binddn", config.Ldap.BindDn)
	Logger.Debugw("LDAP dial.", "uri", config.Ldap.URI)
	var ldapconn *ldap.Conn
	for try := 0; try < 15; try++ {
		ldapconn, err = ldap.DialURL(config.Ldap.URI)
		if err != nil {
			Logger.Debugw("Retrying LDAP connection in 1s.", "error", err)
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return
	}

	defer ldapconn.Close()
	Logger.Debugw("LDAP simple bind.", "binddn", config.Ldap.BindDn)
	err = ldapconn.Bind(config.Ldap.BindDn, config.Ldap.Password)
	if err != nil {
		return
	}

	Logger.Debugw("Running LDAP whoami.")
	wai, err := ldapconn.WhoAmI(nil)
	if err != nil {
		return
	}
	Logger.Debugw("LDAP whoami done.", "authzid", wai.AuthzID)
	return
}
