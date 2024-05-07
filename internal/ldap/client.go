package ldap

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dalibo/ldap2pg/internal/perf"
	ldap3 "github.com/go-ldap/ldap/v3"
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
	uris := options.GetStrings("URI")
	if len(uris) == 0 {
		err = fmt.Errorf("missing URI")
		return
	}

	t := tls.Config{
		InsecureSkipVerify: options.GetString("TLS_REQCERT") != "try",
	}
	d := net.Dialer{
		Timeout: options.GetSeconds("NETWORK_TIMEOUT"),
	}
	try := 0
	err = retry.Do(
		func() error {
			// Round-robin URIs
			i := try % len(uris)
			try++
			client.URI = uris[i]
			slog.Debug("LDAP dial.", "uri", client.URI, "try", try)
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
		if client.BindDN == "" {
			err = fmt.Errorf("missing BINDDN")
			return
		}
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

	slog.Info("Connected to LDAP directory.", "uri", client.URI)
	return
}

func (c *Client) Search(watch *perf.StopWatch, base string, scope Scope, filter string, attributes []string) (*ldap3.SearchResult, error) {
	search := ldap3.SearchRequest{
		BaseDN:     base,
		Scope:      int(scope),
		Filter:     filter,
		Attributes: attributes,
	}
	args := []string{"-b", search.BaseDN, "-s", scope.String(), search.Filter}
	args = append(args, search.Attributes...)
	slog.Debug("Searching LDAP directory.", "cmd", c.Command("ldapsearch", args...))
	var err error
	var res *ldap3.SearchResult
	duration := watch.TimeIt(func() {
		res, err = c.Conn.Search(&search)
	})
	if err != nil {
		slog.Debug("LDAP search failed.", "duration", duration, "err", err)
		return nil, err
	}
	slog.Debug("LDAP search done.", "duration", duration, "entries", len(res.Entries))
	return res, nil
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
