package role_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/role"
	"github.com/stretchr/testify/require"
)

func TestMissingParents(t *testing.T) {
	r := require.New(t)

	current := role.Role{
		Name: "toto",
		Parents: []role.Membership{
			{Name: "parent1"},
		},
	}
	wanted := role.Role{
		Name: "toto",
		Parents: []role.Membership{
			{Name: "parent1"},
			{Name: "parent2"},
		},
	}

	missing := current.MissingParents(wanted.Parents)
	r.Len(missing, 1)
}
