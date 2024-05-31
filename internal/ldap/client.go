package ldap

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dalibo/ldap2pg/internal/perf"
	ldap3 "github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"
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

var Watch perf.StopWatch

func Connect() (client Client, err error) {
	uri := k.String("URI")
	uris := strings.Split(uri, " ")
	if len(uris) == 0 {
		err = fmt.Errorf("missing URI")
		return
	}

	t := tls.Config{
		InsecureSkipVerify: k.String("TLS_REQCERT") != "try",
	}
	d := net.Dialer{
		Timeout: k.Duration("NETWORK_TIMEOUT") * time.Second,
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
				ldap3.DialWithTLSConfig(&t),
				ldap3.DialWithDialer(&d),
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

	client.Timeout = k.Duration("TIMEOUT") * time.Second
	slog.Debug("LDAP set timeout.", "timeout", client.Timeout)
	client.Conn.SetTimeout(client.Timeout)

	client.SaslMech = k.String("SASL_MECH")
	switch client.SaslMech {
	case "":
		client.BindDN = k.String("BINDDN")
		if client.BindDN == "" {
			err = fmt.Errorf("missing BINDDN")
			return
		}
		password := k.String("PASSWORD")
		client.Password = "*******"
		slog.Debug("LDAP simple bind.", "binddn", client.BindDN)
		err = client.Conn.Bind(client.BindDN, password)
	case "DIGEST-MD5":
		client.SaslAuthCID = k.String("SASL_AUTHCID")
		password := k.String("PASSWORD")
		var parsedURI *url.URL
		parsedURI, err = url.Parse(client.URI)
		if err != nil {
			return client, err
		}
		slog.Debug("LDAP SASL/DIGEST-MD5 bind.", "authcid", client.SaslAuthCID, "host", parsedURI.Host)
		err = client.Conn.MD5Bind(parsedURI.Host, client.SaslAuthCID, password)
	case "GSSAPI":
		// Get the principal
		client.SaslAuthCID = k.String("SASL_AUTHCID")
		ccache, ok := os.LookupEnv("KRB5CCNAME")
		if ok {
			ccache = strings.TrimPrefix(ccache, "FILE:")
		} else {
			uid := os.Getuid()
			ccache = fmt.Sprintf("/tmp/krb5cc_%d", uid)
		}
		krb5confPath, ok := os.LookupEnv("KRB5_CONFIG")
		if !ok {
			krb5confPath = "/etc/krb5.conf"
		}
		slog.Debug("Initial SSPI client.", "ccache", ccache, "krb5conf", krb5confPath)
		sspiClient, err := gssapi.NewClientFromCCache(ccache, krb5confPath)
		if err != nil {
			return client, err
		}
		defer sspiClient.Close()
		// Build service Principal from URI.
		var parsedURI *url.URL
		parsedURI, err = url.Parse(client.URI)
		if err != nil {
			return client, err
		}
		spn := "ldap/" + strings.Split(parsedURI.Host, ":")[0]
		slog.Debug("LDAP SASL/GSSAPI bind.", "principal", client.SaslAuthCID, "spn", spn)
		err = client.Conn.GSSAPIBind(sspiClient, spn, client.SaslAuthCID)
		if err != nil {
			return client, err
		}
	default:
		err = fmt.Errorf("unhandled SASL_MECH")
	}
	if err != nil {
		return
	}

	slog.Info("Connected to LDAP directory.", "uri", client.URI)
	return
}

func (c *Client) Search(base string, scope Scope, filter string, attributes []string) (*ldap3.SearchResult, error) {
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
	duration := Watch.TimeIt(func() {
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
