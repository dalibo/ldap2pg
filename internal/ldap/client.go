package ldap

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/avast/retry-go"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

func Connect(options OptionsMap) (conn *ldap3.Conn, err error) {
	uri := options.GetString("URI")
	binddn := options.GetString("BINDDN")

	t := tls.Config{
		InsecureSkipVerify: options.GetString("TLS_REQCERT") != "try",
	}
	d := net.Dialer{
		Timeout: options.GetSeconds("NETWORK_TIMEOUT"),
	}
	slog.Debug("LDAP dial.", "uri", uri)
	err = retry.Do(
		func() error {
			conn, err = ldap3.DialURL(
				uri,
				ldap3.DialWithTLSDialer(&t, &d),
			)
			return err
		},
		retry.RetryIf(IsErrorRecoverable),
		retry.OnRetry(LogRetryError),
		retry.MaxDelay(30*time.Second),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return
	}

	conn.SetTimeout(options.GetSeconds("TIMEOUT"))

	switch options.GetString("SASL_MECH") {
	case "":
		password := options.GetSecret("PASSWORD")
		slog.Debug("LDAP simple bind.", "binddn", binddn)
		err = conn.Bind(binddn, password)
	case "DIGEST-MD5":
		user := options.GetString("SASL_AUTHCID")
		password := options.GetSecret("PASSWORD")
		var parsedURI *url.URL
		parsedURI, err = url.Parse(uri)
		if err != nil {
			return nil, err
		}
		slog.Debug("LDAP SASL/DIGEST-MD5 bind.", "username", user, "host", parsedURI.Host)
		err = conn.MD5Bind(parsedURI.Host, user, password)
	default:
		err = fmt.Errorf("unhandled SASL_MECH")
	}
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

// Implements retry.RetryIfFunc
func IsErrorRecoverable(err error) bool {
	ldapErr, ok := err.(*ldap3.Error)
	if !ok {
		return true
	}
	_, ok = ldapErr.Err.(*tls.CertificateVerificationError)
	// Retrying don't fix bad certificate
	return !ok
}

// Implements retry.OnRetryFunc
func LogRetryError(n uint, err error) {
	slog.Debug("Retrying.", "err", err.Error(), "attempt", n)
}
