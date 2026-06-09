package role_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/role"
	"github.com/stretchr/testify/require"
)

func TestFlatten(t *testing.T) {
	r := require.New(t)

	m := make(role.Map)
	m["group"] = role.Role{Name: "group"}
	m["member"] = role.Role{Name: "member", Parents: []role.Membership{{Name: "group"}}}
	names := m.Flatten()
	r.Len(names, 2)
	r.Equal("group", names[0])
	r.Equal("member", names[1])
}
