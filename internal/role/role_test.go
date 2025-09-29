package role_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/role"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	r := require.New(t)

	r0 := role.New()
	r1 := role.New()
	r0.Merge(r1)

	r1.Config["a"] = "toto"
	r1.Config["b"] = "bobo"
	r0.Merge(r1)
	r.Equal("toto", r0.Config["a"])
	r.Equal("bobo", r0.Config["b"])

	r1.Config["a"] = "tata"
	r0.Merge(r1)
	r.Equal("tata", r0.Config["a"])
}
