package ldap

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Build service Principal from URI.
func buildServicePrincipalName(uri *url.URL) string {
	return "ldap/" + strings.Split(uri.Host, ":")[0]
}

func getCCache() string {
	ccache, ok := os.LookupEnv("KRB5CCNAME")
	if ok {
		return strings.TrimPrefix(ccache, "FILE:")
	}
	uid := os.Getuid()
	return fmt.Sprintf("/tmp/krb5cc_%d", uid)
}

func getKrb5Config() string {
	krb5confPath, ok := os.LookupEnv("KRB5_CONFIG")
	if !ok {
		krb5confPath = "/etc/krb5.conf"
	}
	return krb5confPath
}
