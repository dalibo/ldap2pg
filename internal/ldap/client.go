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

type Client struct {
	URI         string
	BindDN      string
	SaslMech    string
	SaslAuthCID string
	Timeout     time.Duration
	Password    string
	Conn        *ldap3.Conn
}

func Connect(options OptionsMap) (client Client, err error) {
	client.URI = options.GetString("URI")

	t := tls.Config{
		InsecureSkipVerify: options.GetString("TLS_REQCERT") != "try",
	}
	d := net.Dialer{
		Timeout: options.GetSeconds("NETWORK_TIMEOUT"),
	}
	slog.Debug("LDAP dial.", "uri", client.URI)
	err = retry.Do(
		func() error {
			client.Conn, err = ldap3.DialURL(
				client.URI,
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

	client.Timeout = options.GetSeconds("TIMEOUT")
	client.Conn.SetTimeout(client.Timeout)

	client.SaslMech = options.GetString("SASL_MECH")
	switch client.SaslMech {
	case "":
		client.BindDN = options.GetString("BINDDN")
		password := options.GetSecret("PASSWORD")
		client.Password = "*******"
		slog.Debug("LDAP simple bind.", "binddn", client.BindDN)
		err = client.Conn.Bind(client.BindDN, password)
	case "DIGEST-MD5":
		client.SaslAuthCID = options.GetString("SASL_AUTHCID")
		password := options.GetSecret("PASSWORD")
		var parsedURI *url.URL
		parsedURI, err = url.Parse(client.URI)
		if err != nil {
			return client, err
		}
		slog.Debug("LDAP SASL/DIGEST-MD5 bind.", "authcid", client.SaslAuthCID, "host", parsedURI.Host)
		err = client.Conn.MD5Bind(parsedURI.Host, client.SaslAuthCID, password)
	default:
		err = fmt.Errorf("unhandled SASL_MECH")
	}
	if err != nil {
		return
	}

	slog.Debug("Running LDAP whoami.", "cmd", client.Command("ldapwhoami"))
	wai, err := client.Conn.WhoAmI(nil)
	if err != nil {
		return
	}
	slog.Info("Connected to LDAP directory.", "uri", client.URI, "authzid", wai.AuthzID)
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
