package lists_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/lists"
	"github.com/stretchr/testify/require"
)

func TestBlacklist(t *testing.T) {
	r := require.New(t)
	bl := lists.Blacklist{"pif", "paf*"}
	r.Equal("", bl.MatchString("pouf"))
	r.Equal("paf*", bl.MatchString("paf"))
}

func TestBlacklistError(t *testing.T) {
	r := require.New(t)
	// filepath fails if pattern has bad escaping.
	bl := lists.Blacklist{"\\"}
	r.Error(bl.Check())
}
