package ldap

import (
	"github.com/avast/retry-go"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

func Connect(options OptionsMap) (conn *ldap.Conn, err error) {
	uri := options.GetString("URI")
	binddn := options.GetString("BINDDN")

	slog.Debug("LDAP dial.", "uri", uri)
	err = retry.Do(func() error {
		conn, err = ldap.DialURL(uri)
		return err
	})
	if err != nil {
		return
	}

	slog.Debug("LDAP simple bind.", "binddn", binddn)
	err = conn.Bind(binddn, options.GetString("PASSWORD"))
	if err != nil {
		return
	}

	slog.Debug("Running LDAP whoami.")
	wai, err := conn.WhoAmI(nil)
	if err != nil {
		return
	}
	slog.Info("Connected to LDAP directory.", "uri", uri, "authzid", wai.AuthzID)
	return
}
