package tree_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/tree"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	r := require.New(t)
	groups := map[string][]string{
		// usage is before select, must be sorted.
		"ro": {"__connect__", "__usage__", "__select__"},
		// subgroup is before rw, must be sorted.
		"subgroup": {"__select__", "__usage__"},
		"rw":       {"ro"},
		"ddl":      {"rw"},
	}
	order := tree.Walk(groups)
	wanted := []string{"__connect__", "__select__", "__usage__", "ro", "rw", "ddl", "subgroup"}
	r.Equal(wanted, order)
}
