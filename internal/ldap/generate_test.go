package ldap_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/ldap"
	"github.com/stretchr/testify/require"
)

func TestResolveRDNUpper(t *testing.T) {
	r := require.New(t)

	attrValues := map[string]string{
		"member": "CN=Alice,OU=Users,DC=bridoulou,DC=fr",
	}
	expressions := []string{"member.cn"}
	result := &ldap.Result{}

	exprMap := result.ResolveExpressions(expressions, attrValues, nil)
	r.Equal("Alice", exprMap["member.cn"])
}
