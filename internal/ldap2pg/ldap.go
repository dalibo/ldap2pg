package ldap2pg

import (
	"time"

	ldap "github.com/go-ldap/ldap/v3"
	log "github.com/sirupsen/logrus"
)

func LdapConnect(config Config) (err error) {
	log.
		WithField("uri", config.Ldap.URI).
		WithField("binddn", config.Ldap.BindDn).
		Info("Connecting to LDAP directory.")
	log.
		WithField("uri", config.Ldap.URI).
		Debug("LDAP dial.")
	var ldapconn *ldap.Conn
	for try := 0; try < 15; try++ {
		ldapconn, err = ldap.DialURL(config.Ldap.URI)
		if err != nil {
			log.
				WithField("error", err).
				Debug("Retrying LDAP connection in 1s.")
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return
	}

	defer ldapconn.Close()
	log.
		WithField("binddn", config.Ldap.BindDn).
		Debug("LDAP simple bind.")
	err = ldapconn.Bind(config.Ldap.BindDn, config.Ldap.Password)
	if err != nil {
		return
	}

	log.Debug("Running LDAP whoami.")
	wai, err := ldapconn.WhoAmI(nil)
	if err != nil {
		return
	}
	log.
		WithField("authzid", wai.AuthzID).
		Debug("LDAP whoami done.")
	return
}
